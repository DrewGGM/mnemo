import hashlib
import json
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import pytest

from mnemo_eval import pipeline
from mnemo_eval.client import MnemoClient, MnemoError

SMOKE = Path(__file__).resolve().parents[1] / "config" / "smoke.yaml"
RESULT_FIELDS = {
    "protocol",
    "question_id",
    "type",
    "answer",
    "correct",
    "reader_input_tokens",
    "recall_ms",
}


class FakeMnemo:
    """Records what is remembered and recalls fixed contents per question id."""

    def __init__(self, recalled):
        self.recalled = recalled
        self.remembered = []

    def remember(self, subject, session_id, messages, wait=True):
        self.remembered.append((subject, session_id, messages, wait))
        return {"episode_ids": []}

    def recall(self, subject, query, **params):
        qid = next(q for q in self.recalled if subject.endswith("-" + q))
        return {"episodes": [{"content": c} for c in self.recalled[qid]]}


def run_smoke(tmp_path, recalled):
    spec = pipeline.load_protocol(SMOKE)
    arms = pipeline.select_arms(spec, ["all"])
    questions = pipeline.load_questions(spec, None)
    paths = pipeline.run(spec, arms, questions, FakeMnemo(recalled), tmp_path)
    rows = {
        arm: [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
        for arm, path in paths.items()
    }
    return spec, paths, rows


def test_correct_reader_answer_counts_as_hit(tmp_path):
    recalled = {"toy-dog": ["My dog is called Toby."], "toy-city": ["I moved to Medellin."]}
    _, _, rows = run_smoke(tmp_path, recalled)
    assert [r["correct"] for r in rows["B2"]] == [True, True]


def test_wrong_reader_answer_counts_as_miss(tmp_path):
    recalled = {"toy-dog": ["He turned three in January."], "toy-city": ["I live in Bogota."]}
    _, _, rows = run_smoke(tmp_path, recalled)
    assert [r["correct"] for r in rows["B2"]] == [False, False]


def test_arm_without_memory_does_not_know(tmp_path):
    _, _, rows = run_smoke(tmp_path, {"toy-dog": ["Toby"], "toy-city": ["Medellin"]})
    assert [(r["answer"], r["correct"], r["recall_ms"]) for r in rows["B0"]] == [
        (pipeline.NO_ANSWER, False, None)
    ] * 2


def test_protocol_id_is_sha256_of_yaml(tmp_path):
    spec, _, _ = run_smoke(tmp_path, {"toy-dog": [], "toy-city": []})
    assert spec.id == hashlib.sha256(SMOKE.read_bytes()).hexdigest()


def test_every_result_line_carries_protocol_id(tmp_path):
    spec, _, rows = run_smoke(tmp_path, {"toy-dog": [], "toy-city": []})
    assert {r["protocol"] for arm in rows.values() for r in arm} == {spec.id}


def test_one_file_per_arm_under_protocol_id(tmp_path):
    spec, paths, _ = run_smoke(tmp_path, {"toy-dog": [], "toy-city": []})
    assert paths == {arm: tmp_path / spec.id / f"{arm}.jsonl" for arm in ("B0", "B2")}


def test_result_line_fields(tmp_path):
    _, _, rows = run_smoke(tmp_path, {"toy-dog": ["Toby"], "toy-city": []})
    for row in rows["B0"] + rows["B2"]:
        assert set(row) == RESULT_FIELDS
        assert row["reader_input_tokens"] > 0
    assert all(r["recall_ms"] >= 0 for r in rows["B2"])


def test_ingest_sends_sessions_in_date_order():
    question = {
        "sessions": [
            {"session_id": "late", "date": "2026-02-01T00:00:00Z", "messages": [{"role": "user"}]},
            {"session_id": "early", "date": "2026-01-01T00:00:00Z", "messages": [{"role": "user"}]},
        ]
    }
    fake = FakeMnemo({})
    pipeline.ingest(fake, "s", question)
    assert [(sid, msgs[0]["occurred_at"], wait) for _, sid, msgs, wait in fake.remembered] == [
        ("early", "2026-01-01T00:00:00Z", True),
        ("late", "2026-02-01T00:00:00Z", True),
    ]


def write_protocol(tmp_path, body):
    path = tmp_path / "protocol.yaml"
    dataset = SMOKE.parent.parent / "fixtures" / "toy.json"
    path.write_text(f"dataset: {dataset.as_posix()}\n{body}", encoding="utf-8")
    return path


def test_split_keeps_only_its_questions(tmp_path):
    body = "reader: fake\njudge: fake\narms: {}\nsplits:\n  one: [toy-city]\n"
    spec = pipeline.load_protocol(write_protocol(tmp_path, body))
    assert [q["question_id"] for q in pipeline.load_questions(spec, "one")] == ["toy-city"]


def test_unknown_split_is_rejected(tmp_path):
    spec = pipeline.load_protocol(write_protocol(tmp_path, "reader: fake\njudge: fake\narms: {}\n"))
    with pytest.raises(ValueError, match="unknown split"):
        pipeline.load_questions(spec, "dev60")


def test_real_reader_is_rejected(tmp_path):
    body = "reader: gemini\njudge: fake\narms: {}\n"
    with pytest.raises(ValueError, match="reader must be 'fake'"):
        pipeline.load_protocol(write_protocol(tmp_path, body))


def test_arm_name_outside_runs_dir_is_rejected(tmp_path):
    body = "reader: fake\njudge: fake\narms:\n  ../B0:\n    memory: none\n"
    with pytest.raises(ValueError, match="arm name"):
        pipeline.load_protocol(write_protocol(tmp_path, body))


class RedirectHandler(BaseHTTPRequestHandler):
    """Answers every POST with a 302 to /leak and records the paths it receives."""

    def do_POST(self):
        self.server.paths.append(self.path)
        self.send_response(302)
        self.send_header("Location", "/leak")
        self.send_header("Content-Length", "0")
        self.end_headers()

    do_GET = do_POST

    def log_message(self, format, *args):
        pass


def test_client_does_not_follow_redirects():
    server = ThreadingHTTPServer(("127.0.0.1", 0), RedirectHandler)
    server.paths = []
    thread = threading.Thread(target=server.serve_forever)
    thread.start()
    try:
        client = MnemoClient(f"http://127.0.0.1:{server.server_port}", api_key="secret")
        with pytest.raises(MnemoError, match="HTTP 302"):
            client.recall("s", "q")
    finally:
        server.shutdown()
        server.server_close()
        thread.join()
    assert server.paths == ["/v1/recall"]
