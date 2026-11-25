import hashlib
import json
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import pytest

from mnemo_eval import arms, judge, models, pipeline, report
from mnemo_eval.client import MnemoClient, MnemoError
from mnemo_eval.datasets.longmemeval import Question, Session

CONFIG = Path(__file__).resolve().parents[1] / "config"
SMOKE = CONFIG / "smoke.yaml"
PROTOCOL = CONFIG / "protocol.yaml"
RESULT_FIELDS = {
    "protocol",
    "question_id",
    "type",
    "is_abstention",
    "answer",
    "correct",
    "reader_input_tokens",
    "reader_output_tokens",
    "context_tokens",
    "recall_ms",
    "retrieved_session_ids",
    "evidence_session_ids",
}


class FakeMnemo:
    """Records what is remembered and recalls fixed contents per question id."""

    def __init__(self, recalled):
        self.recalled = recalled
        self.remembered = []
        self.recalls = []

    def remember(self, subject, session_id, messages, wait=True):
        self.remembered.append((subject, session_id, messages, wait))
        return {"episode_ids": []}

    def recall(self, subject, query, **params):
        self.recalls.append(params)
        qid = next(q for q in self.recalled if subject.endswith("-" + q))
        return {"episodes": [episode(c) for c in self.recalled[qid]]}


def episode(content, session_id="s1", role="user"):
    return {
        "content": content,
        "session_id": session_id,
        "role": role,
        "occurred_at": "2026-03-10T15:04:00Z",
    }


def run_smoke(tmp_path, recalled):
    spec = pipeline.load_protocol(SMOKE)
    arm_cfgs = pipeline.select_arms(spec, ["all"])
    questions = pipeline.load_questions(spec, None)
    backends = pipeline.open_backends(spec, FakeMnemo(recalled), {})
    paths = pipeline.run(spec, arm_cfgs, questions, backends, tmp_path)
    rows = {arm: pipeline.read_rows(path) for arm, path in paths.items()}
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
        (models.NO_ANSWER, False, None)
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
    assert rows["B2"][0]["retrieved_session_ids"] == ["s1"]


def toy_question(sessions, qid="q1", abstention=False, qtype="multi-session"):
    return Question(
        id=qid,
        type=qtype,
        question="What is my dog called?",
        answer="Toby",
        question_date="2023/05/30 (Tue) 23:40",
        sessions=sessions,
        evidence_session_ids=["late"],
        is_abstention=abstention,
    )


def session(sid, date, *contents):
    messages = [{"role": "user", "content": c} for c in contents]
    return Session(id=sid, date=date, messages=messages, evidence_turns=[])


def test_ingest_sends_sessions_in_date_order():
    question = toy_question(
        [
            session("late", "2023/05/21 (Sun) 08:00", "b"),
            session("early", "2023/05/20 (Sat) 02:21", "a"),
        ]
    )
    fake = FakeMnemo({})
    pipeline.ingest(fake, "s", question)
    assert [(sid, msgs[0]["occurred_at"], wait) for _, sid, msgs, wait in fake.remembered] == [
        ("early", "2023-05-20T02:21:00Z", True),
        ("late", "2023-05-21T08:00:00Z", True),
    ]


def test_ingest_suffixes_a_session_id_repeated_in_one_history():
    question = toy_question(
        [session("s", "2023/05/20 (Sat) 02:21", "a"), session("s", "2023/05/21 (Sun) 08:00", "a")]
    )
    fake = FakeMnemo({})
    pipeline.ingest(fake, "s", question)
    assert [sid for _, sid, _, _ in fake.remembered] == ["s", "s#dup1"]


def test_ingest_leaves_out_empty_messages_and_splits_long_ones():
    # The leading "a" puts the byte limit inside a two-byte character.
    long = "a" + "é" * (pipeline.MAX_CONTENT_BYTES // 2)
    question = toy_question([session("s", "2023/05/20 (Sat) 02:21", "", long)])
    fake = FakeMnemo({})
    pipeline.ingest(fake, "s", question)
    contents = [m["content"] for m in fake.remembered[0][2]]
    assert "".join(contents) == long
    assert all(0 < len(c.encode()) <= pipeline.MAX_CONTENT_BYTES for c in contents)


def test_full_history_arm_is_chronological_without_session_ids():
    question = toy_question(
        [
            session("answer_late", "2023/05/21 (Sun) 08:00", "My dog is Toby."),
            session("early", "2023/05/20 (Sat) 02:21", "Hi."),
        ]
    )
    cfg = arms.ArmConfig("B1", "history", {})
    out = arms.full_history(question, cfg, FakeMnemo({}), "s")
    assert out.context == (
        "[2023-05-20 (Sat) 02:21] user: Hi.\n[2023-05-21 (Sun) 08:00] user: My dog is Toby."
    )
    assert out.context_tokens == (len(out.context) + 3) // 4
    assert (out.recall_ms, out.retrieved_session_ids) == (None, [])


def test_recall_arm_sends_protocol_params_and_strips_duplicate_suffix():
    fake = FakeMnemo({"q1": []})
    fake.recall = lambda subject, query, **params: (
        fake.recalls.append(params) or {"episodes": [episode("x", "s#dup1"), episode("y", "t")]}
    )
    cfg = arms.ArmConfig("B2", "recall", {"sources": ["episodes.vector"], "max_tokens": 8000})
    out = arms.recall(toy_question([]), cfg, fake, "eval-q1")
    assert fake.recalls == [{"sources": ["episodes.vector"], "max_tokens": 8000}]
    assert out.retrieved_session_ids == ["s", "t"]
    assert out.context.splitlines()[0] == "[2026-03-10 (Tue) 15:04] user: x"


def test_protocol_arms_follow_the_evaluation_plan():
    spec = pipeline.load_protocol(PROTOCOL)
    assert {n: (a.memory, a.recall) for n, a in spec.arms.items()} == {
        "B0": ("none", {}),
        "B1": ("history", {}),
        "B2": ("recall", {"sources": ["episodes.vector"], "max_tokens": 8000}),
        "M": ("recall", {"max_tokens": 8000}),
    }


def test_every_arm_reads_with_the_same_prompt(tmp_path):
    spec = pipeline.load_protocol(SMOKE)
    prompts = []

    class Recorder(models.FakeReader):
        def complete(self, prompt, max_tokens=None):
            prompts.append(prompt)
            return super().complete(prompt, max_tokens)

    mnemo = FakeMnemo({"toy-dog": ["Toby"]})
    backends = pipeline.Backends(mnemo=mnemo, reader=Recorder(), judge=models.FakeJudge())
    questions = pipeline.load_questions(spec, None)[:1]
    pipeline.run(spec, list(spec.arms.values()), questions, backends, tmp_path)
    template = pipeline.READER_PROMPT.read_text(encoding="utf-8")
    head = template.split("{memory}")[0]
    assert [p.startswith(head) for p in prompts] == [True, True]
    assert all("Question: What is my dog called?" in p for p in prompts)
    date = arms.show_date(questions[0].question_date)
    assert all(f"Current date: {date}" in p for p in prompts)


def test_arm_named_twice_is_rejected():
    spec = pipeline.load_protocol(SMOKE)
    with pytest.raises(ValueError, match="more than once"):
        pipeline.select_arms(spec, ["B0", "B0"])


def write_protocol(tmp_path, body):
    path = tmp_path / "protocol.yaml"
    dataset = CONFIG.parent / "fixtures" / "toy.json"
    path.write_text(f"dataset: {dataset.as_posix()}\n{body}", encoding="utf-8")
    return path


def test_split_keeps_only_its_questions(tmp_path):
    body = "reader: fake\njudge: fake\narms: {}\nsplits:\n  one: [toy-city]\n"
    spec = pipeline.load_protocol(write_protocol(tmp_path, body))
    assert [q.id for q in pipeline.load_questions(spec, "one")] == ["toy-city"]


def test_unknown_split_is_rejected(tmp_path):
    spec = pipeline.load_protocol(write_protocol(tmp_path, "reader: fake\njudge: fake\narms: {}\n"))
    with pytest.raises(ValueError, match="unknown split"):
        pipeline.load_questions(spec, "dev60")


def test_real_reader_must_pin_the_reader_prompt(tmp_path):
    body = "reader: {model: m, env: MNEMO_LLM, prompt_sha256: x}\njudge: fake\narms: {}\n"
    with pytest.raises(ValueError, match="prompt_sha256"):
        pipeline.load_protocol(write_protocol(tmp_path, body))


def test_real_model_needs_its_base_url():
    spec = pipeline.load_protocol(PROTOCOL)
    with pytest.raises(ValueError, match="MNEMO_LLM_BASE_URL is not set"):
        pipeline.open_backends(spec, FakeMnemo({}), {})


def test_env_model_other_than_protocol_is_rejected():
    spec = pipeline.load_protocol(PROTOCOL)
    env = {"MNEMO_LLM_BASE_URL": "https://llm.test/v1", "MNEMO_LLM_MODEL": "other"}
    with pytest.raises(ValueError, match="protocol fixes 'gemini-3.1-flash-lite'"):
        pipeline.open_backends(spec, FakeMnemo({}), env)


def test_arm_name_outside_runs_dir_is_rejected(tmp_path):
    body = "reader: fake\njudge: fake\narms:\n  ../B0:\n    memory: none\n"
    with pytest.raises(ValueError, match="arm name"):
        pipeline.load_protocol(write_protocol(tmp_path, body))


@pytest.mark.parametrize(
    ("qtype", "abstention", "phrase"),
    [
        ("temporal-reasoning", False, "do not penalize off-by-one errors"),
        ("knowledge-update", False, "along with an updated answer"),
        ("single-session-preference", False, "Rubric: Toby"),
        ("multi-session", False, "Correct Answer: Toby"),
        ("multi-session", True, "identifies the question as unanswerable"),
    ],
)
def test_judge_prompt_follows_question_type(qtype, abstention, phrase):
    text = judge.prompt(toy_question([], abstention=abstention, qtype=qtype), "It is Toby.")
    assert phrase in text
    assert text.endswith("Answer yes or no only.")


def test_judge_rejects_unknown_type():
    with pytest.raises(ValueError, match="no judge prompt"):
        judge.prompt(toy_question([], qtype="other"), "x")


def test_judge_reads_yes_in_lower_case_with_ten_tokens():
    asked = []

    class Says:
        def __init__(self, text):
            self.text = text

        def complete(self, prompt, max_tokens=None):
            asked.append(max_tokens)
            return models.Completion(self.text, 1, 1)

    question = toy_question([])
    assert judge.judge(question, "Toby", Says("Yes."))
    assert not judge.judge(question, "Rex", Says("No"))
    assert asked == [10, 10]


def test_report_writes_summary_and_table(tmp_path):
    spec, _, _ = run_smoke(tmp_path, {"toy-dog": ["My dog is called Toby."], "toy-city": []})
    summary = report.summarize(spec, report.load_runs(spec, tmp_path))
    json_path, md_path = report.write(summary, tmp_path / "results")
    saved = json.loads(json_path.read_text(encoding="utf-8"))
    assert saved["protocol"] == spec.id
    assert saved["arms"]["B2"]["accuracy"] == pytest.approx(0.5)
    assert saved["arms"]["B0"]["recall_ms"] is None
    assert "| B2 | 2 | 50.0%" in md_path.read_text(encoding="utf-8")


def test_report_refuses_arms_run_on_different_questions(tmp_path):
    spec = pipeline.load_protocol(SMOKE)
    (tmp_path / spec.id).mkdir()
    for arm, qid in (("B0", "toy-dog"), ("B2", "toy-city")):
        pipeline.append(tmp_path / spec.id / f"{arm}.jsonl", {"question_id": qid})
    with pytest.raises(ValueError, match="different questions"):
        report.load_runs(spec, tmp_path)


def write_rows(path, *protocols):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("".join(json.dumps({"protocol": p}) + "\n" for p in protocols), "utf-8")


def test_verify_protocol_accepts_one_protocol_per_directory(tmp_path):
    write_rows(tmp_path / "a" / "B0.jsonl", "a", "a")
    write_rows(tmp_path / "b" / "B0.jsonl", "b")
    assert report.verify_protocol(tmp_path) == (2, [])


def test_verify_protocol_rejects_mixed_protocol_ids(tmp_path):
    write_rows(tmp_path / "a" / "B0.jsonl", "a")
    write_rows(tmp_path / "a" / "B2.jsonl", "a", "b")
    _, problems = report.verify_protocol(tmp_path)
    assert problems == [f"{tmp_path / 'a'}: results of 2 protocols: ['a', 'b']"]


def test_verify_protocol_rejects_lines_without_protocol(tmp_path):
    (tmp_path / "B0.jsonl").write_text('{"question_id": "q"}\n', encoding="utf-8")
    _, problems = report.verify_protocol(tmp_path)
    assert problems == [f"{tmp_path / 'B0.jsonl'}:1: no protocol id"]


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
