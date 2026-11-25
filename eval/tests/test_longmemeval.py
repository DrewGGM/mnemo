import hashlib
import json
import threading
from collections import Counter
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import pytest

from mnemo_eval.datasets import longmemeval as lme

PROTOCOL = Path(__file__).resolve().parents[1] / "config" / "protocol.yaml"
SEED = 7
# Question counts per type of LongMemEval_S, abstention questions apart.
TYPE_COUNTS = {
    "knowledge-update": 72,
    "knowledge-update_abs": 6,
    "multi-session": 121,
    "multi-session_abs": 12,
    "single-session-assistant": 56,
    "single-session-preference": 30,
    "single-session-user": 64,
    "single-session-user_abs": 6,
    "temporal-reasoning": 127,
    "temporal-reasoning_abs": 6,
}
TOTAL = 500
ABSTENTION = 30
DEV = 60
WRITE = 100
# Questions whose history has a session id twice.
REPEATED_ID_HISTORIES = 13


def record(qid, qtype="single-session-user", sessions=(("s1", "Hi"),)):
    return {
        "question_id": qid,
        "question_type": qtype,
        "question": f"What about {qid}?",
        "answer": 3,
        "question_date": "2023/05/30 (Tue) 23:40",
        "haystack_session_ids": [sid for sid, _ in sessions],
        "haystack_dates": ["2023/05/20 (Sat) 02:21"] * len(sessions),
        "haystack_sessions": [
            [{"role": "user", "content": text, "has_answer": True}] for _, text in sessions
        ],
        "answer_session_ids": [sessions[0][0]],
    }


def write_records(tmp_path, records):
    path = tmp_path / "longmemeval.json"
    path.write_text(json.dumps(records), encoding="utf-8")
    return path


def synthetic_questions(tmp_path):
    records = []
    for key, n in TYPE_COUNTS.items():
        qtype = key.removesuffix("_abs")
        suffix = "_abs" if key.endswith("_abs") else ""
        records += [record(f"{qtype}-{i}{suffix}", qtype) for i in range(n)]
    return lme.load_longmemeval(write_records(tmp_path, records))


def strata(questions):
    return Counter(q.type + ("_abs" if q.is_abstention else "") for q in questions)


def test_load_maps_fields(tmp_path):
    rec = record("q1_abs", sessions=[("a", "Hi")])
    rec["haystack_sessions"][0].insert(0, {"role": "assistant", "content": "Hello"})
    [q] = lme.load_longmemeval(write_records(tmp_path, [rec]))
    assert q == lme.Question(
        id="q1_abs",
        type="single-session-user",
        question="What about q1_abs?",
        answer="3",
        question_date="2023/05/30 (Tue) 23:40",
        sessions=[
            lme.Session(
                id="a",
                date="2023/05/20 (Sat) 02:21",
                messages=[
                    {"role": "assistant", "content": "Hello"},
                    {"role": "user", "content": "Hi"},
                ],
                evidence_turns=[1],
            )
        ],
        evidence_session_ids=["a"],
        is_abstention=True,
    )


def test_load_names_missing_question_field(tmp_path):
    rec = record("q1")
    del rec["haystack_dates"]
    with pytest.raises(ValueError, match="question q1: missing field haystack_dates"):
        lme.load_longmemeval(write_records(tmp_path, [rec]))


def test_load_names_missing_turn_field(tmp_path):
    rec = record("q1")
    del rec["haystack_sessions"][0][0]["content"]
    with pytest.raises(ValueError, match="question q1: missing field content"):
        lme.load_longmemeval(write_records(tmp_path, [rec]))


def test_load_rejects_haystack_of_different_lengths(tmp_path):
    rec = record("q1")
    rec["haystack_dates"].append("2023/05/21 (Sun) 10:00")
    with pytest.raises(ValueError, match="differ in length"):
        lme.load_longmemeval(write_records(tmp_path, [rec]))


def test_dev60_is_stratified_by_type(tmp_path):
    dev = lme.split(synthetic_questions(tmp_path), "dev60", SEED)
    assert strata(dev) == {
        "knowledge-update": 9,
        "knowledge-update_abs": 1,
        "multi-session": 14,
        "multi-session_abs": 1,
        "single-session-assistant": 7,
        "single-session-preference": 3,
        "single-session-user": 8,
        "single-session-user_abs": 1,
        "temporal-reasoning": 15,
        "temporal-reasoning_abs": 1,
    }


def test_split_is_deterministic(tmp_path):
    questions = synthetic_questions(tmp_path)
    first = [q.id for q in lme.split(questions, "dev60", SEED)]
    again = [q.id for q in lme.split(list(reversed(questions)), "dev60", SEED)]
    assert sorted(first) == sorted(again)


def test_split_changes_with_seed(tmp_path):
    questions = synthetic_questions(tmp_path)
    assert lme.split(questions, "dev60", SEED) != lme.split(questions, "dev60", SEED + 1)


def test_dev60_is_disjoint_from_heldout440(tmp_path):
    questions = synthetic_questions(tmp_path)
    dev = {q.id for q in lme.split(questions, "dev60", SEED)}
    held = {q.id for q in lme.split(questions, "heldout440", SEED)}
    assert not dev & held


def test_dev60_and_heldout440_cover_all(tmp_path):
    questions = synthetic_questions(tmp_path)
    dev = lme.split(questions, "dev60", SEED)
    held = lme.split(questions, "heldout440", SEED)
    assert (len(dev), len(held)) == (DEV, TOTAL - DEV)
    assert {q.id for q in dev + held} == {q.id for q in lme.split(questions, "all", SEED)}


def test_write100_is_stratified_sample(tmp_path):
    write = lme.split(synthetic_questions(tmp_path), "write100", SEED)
    assert len(write) == WRITE
    got = strata(write)
    assert all(abs(got[k] - n * WRITE / TOTAL) < 1 for k, n in TYPE_COUNTS.items())


def test_unknown_split_is_rejected(tmp_path):
    with pytest.raises(ValueError, match="unknown split"):
        lme.split(synthetic_questions(tmp_path), "dev61", SEED)


def test_session_repetition_counts_content_across_questions(tmp_path):
    records = [
        record("q1", sessions=[("a", "shared"), ("b", "own")]),
        record("q2", sessions=[("c", "shared"), ("d", "only q2")]),
        record("q3", sessions=[("e", "same twice"), ("f", "same twice")]),
    ]
    rep = lme.session_repetition(lme.load_longmemeval(write_records(tmp_path, records)))
    assert rep == lme.Repetition(sessions=6, repeated=2, distinct=4)


class FileHandler(BaseHTTPRequestHandler):
    """Serves server.body at every path and counts the requests."""

    def do_GET(self):
        self.server.requests += 1
        self.send_response(200)
        self.send_header("Content-Length", str(len(self.server.body)))
        self.end_headers()
        self.wfile.write(self.server.body)

    def log_message(self, format, *args):
        pass


@pytest.fixture
def server():
    srv = ThreadingHTTPServer(("127.0.0.1", 0), FileHandler)
    srv.body = b'[{"question_id": "q1"}]'
    srv.requests = 0
    thread = threading.Thread(target=srv.serve_forever)
    thread.start()
    yield srv
    srv.shutdown()
    srv.server_close()
    thread.join()


def data_file(server, tmp_path, sha256):
    url = f"http://127.0.0.1:{server.server_port}/longmemeval_s_cleaned.json"
    return lme.DataFile(url=url, sha256=sha256, path=tmp_path / "data" / "s.json")


def test_download_rejects_sha256_mismatch(server, tmp_path):
    f = data_file(server, tmp_path, "0" * 64)
    with pytest.raises(ValueError, match="does not match"):
        lme.download(f)
    assert list(f.path.parent.iterdir()) == []


def test_download_keeps_file_with_pinned_sha256(server, tmp_path):
    want = hashlib.sha256(server.body).hexdigest()
    f = data_file(server, tmp_path, want)
    assert lme.download(f) == want
    assert f.path.read_bytes() == server.body


def test_download_rejects_body_over_limit(server, tmp_path, monkeypatch):
    monkeypatch.setattr(lme, "MAX_DOWNLOAD_BYTES", len(server.body) - 1)
    f = data_file(server, tmp_path, "")
    with pytest.raises(ValueError, match="larger than"):
        lme.download(f)
    assert list(f.path.parent.iterdir()) == []


def test_download_skips_file_already_there(server, tmp_path):
    f = data_file(server, tmp_path, hashlib.sha256(server.body).hexdigest())
    lme.download(f)
    lme.download(f)
    assert server.requests == 1


def test_download_without_pinned_sha256_returns_it(server, tmp_path):
    f = data_file(server, tmp_path, "")
    assert lme.download(f) == hashlib.sha256(server.body).hexdigest()
    assert f.path.exists()


def test_protocol_rejects_plain_http(tmp_path):
    path = tmp_path / "protocol.yaml"
    path.write_text(
        "seed: 1\ndata_dir: data\ndatasets:\n  s:\n    url: http://example.com/s.json\n",
        encoding="utf-8",
    )
    with pytest.raises(ValueError, match="https"):
        lme.load_data_protocol(path)


# The real files, downloaded by mnemo-eval fetch; skipped where they are absent (CI).
DATA = lme.load_data_protocol(PROTOCOL)
REAL_S = DATA.files["longmemeval_s"].path
REAL_ORACLE = DATA.files["longmemeval_oracle"].path
needs_data = pytest.mark.skipif(not REAL_S.exists(), reason="run mnemo-eval fetch first")
needs_oracle = pytest.mark.skipif(not REAL_ORACLE.exists(), reason="run mnemo-eval fetch first")


@pytest.fixture(scope="module")
def real():
    return lme.load_longmemeval(REAL_S)


@needs_data
def test_real_file_has_500_questions(real):
    assert len({q.id for q in real}) == TOTAL


@needs_data
def test_real_file_has_30_abstention(real):
    assert sum(q.is_abstention for q in real) == ABSTENTION


@needs_data
def test_real_type_counts(real):
    assert strata(real) == TYPE_COUNTS


@needs_data
def test_real_dev60_is_disjoint_from_heldout440(real):
    dev = {q.id for q in lme.split(real, "dev60", DATA.seed)}
    held = {q.id for q in lme.split(real, "heldout440", DATA.seed)}
    assert (len(dev), len(held), len(dev & held)) == (DEV, TOTAL - DEV, 0)


@needs_data
def test_real_dev60_and_heldout440_cover_all(real):
    dev = {q.id for q in lme.split(real, "dev60", DATA.seed)}
    held = {q.id for q in lme.split(real, "heldout440", DATA.seed)}
    assert dev | held == {q.id for q in real}


@needs_data
def test_real_session_ids_repeat_within_13_histories(real):
    # Same content, another date; pipeline.ingest sends the id as the episode's session_id.
    repeats = [q for q in real if len({s.id for s in q.sessions}) < len(q.sessions)]
    assert len(repeats) == REPEATED_ID_HISTORIES


@needs_data
@needs_oracle
def test_real_oracle_has_same_questions(real):
    oracle = lme.load_longmemeval(REAL_ORACLE)
    assert {q.id for q in oracle} == {q.id for q in real}
