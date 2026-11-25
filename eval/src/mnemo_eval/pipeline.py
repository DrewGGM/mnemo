"""The evaluation pipeline: loads the protocol and its questions, ingests each
history, runs each arm, reads and judges its answer and appends the result rows."""

import hashlib
import json
import os
import re
import secrets
from collections import Counter
from collections.abc import Mapping
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml

from mnemo_eval import arms, judge, models
from mnemo_eval.client import MnemoClient
from mnemo_eval.datasets import longmemeval
from mnemo_eval.datasets.longmemeval import Question, Session

# Arm names become file names under eval/runs.
ARM_NAME = re.compile(r"[A-Za-z0-9_-]+")
READER_PROMPT = Path(__file__).parent / "prompts" / "reader.txt"
# The service refuses empty messages and messages over 32 KiB (0001_init.sql).
MAX_CONTENT_BYTES = 32768
# Bytes 10xxxxxx continue a UTF-8 character.
UTF8_CONTINUATION_MASK, UTF8_CONTINUATION = 0xC0, 0x80


@dataclass(frozen=True)
class ModelSpec:
    """A role of the protocol: the model id and the prefix of its variables."""

    model: str
    env: str


@dataclass(frozen=True)
class Spec:
    """A loaded protocol file; id is the SHA-256 of its bytes."""

    id: str
    path: Path
    arms: dict[str, arms.ArmConfig]
    reader: ModelSpec
    judge: ModelSpec
    reader_prompt: str
    seed: int
    prices: dict[str, Any]
    cfg: dict[str, Any]


@dataclass(frozen=True)
class Backends:
    """What a run talks to: the mnemo service, the reader and the judge."""

    mnemo: MnemoClient
    reader: models.ChatModel
    judge: models.ChatModel


def load_protocol(path: Path) -> Spec:
    """Reads a protocol YAML; dataset paths are relative to the YAML file."""
    raw = path.read_bytes()
    cfg = yaml.safe_load(raw)
    if not isinstance(cfg, dict) or not isinstance(cfg.get("arms"), dict):
        raise ValueError(f"{path}: expected a mapping with an arms mapping")
    reader, judge_spec = _model(path, cfg, "reader"), _model(path, cfg, "judge")
    # Newlines normalized: a Windows checkout must not change the pinned hash.
    prompt = READER_PROMPT.read_text(encoding="utf-8")
    if reader.model != models.FAKE:
        _check_prompt(path, cfg["reader"], prompt)
    return Spec(
        id=hashlib.sha256(raw).hexdigest(),
        path=path,
        arms={name: _arm(name, a) for name, a in cfg["arms"].items()},
        reader=reader,
        judge=judge_spec,
        reader_prompt=prompt,
        seed=int(cfg.get("seed", 0)),
        prices=cfg.get("prices", {}),
        cfg=cfg,
    )


def _model(path: Path, cfg: dict[str, Any], role: str) -> ModelSpec:
    """A role is "fake" or a mapping with model and env."""
    value = cfg.get(role)
    if value == models.FAKE:
        return ModelSpec(models.FAKE, "")
    if not isinstance(value, dict) or not value.get("model") or not value.get("env"):
        raise ValueError(f"{path}: {role} must be {models.FAKE!r} or a mapping with model and env")
    return ModelSpec(str(value["model"]), str(value["env"]))


def _check_prompt(path: Path, reader: dict[str, Any], prompt: str) -> None:
    """A real reader pins the reader prompt, so that the protocol id covers it."""
    got = hashlib.sha256(prompt.encode()).hexdigest()
    if reader.get("prompt_sha256") != got:
        raise ValueError(
            f"{path}: reader prompt_sha256 must be the SHA-256 of {READER_PROMPT.name}, {got}"
        )


def _arm(name: str, cfg: dict[str, Any]) -> arms.ArmConfig:
    if not ARM_NAME.fullmatch(str(name)):
        raise ValueError(f"arm name {name!r} must use only letters, digits, - and _")
    memory = cfg["memory"]
    if memory not in arms.MEMORY_KINDS:
        raise ValueError(f"arm {name}: memory must be one of {arms.MEMORY_KINDS}, got {memory!r}")
    return arms.ArmConfig(name=name, memory=memory, recall=cfg.get("recall", {}))


def select_arms(spec: Spec, names: list[str]) -> list[arms.ArmConfig]:
    """Returns the arms named, or every arm for ["all"]."""
    if names == ["all"]:
        return list(spec.arms.values())
    if len(set(names)) != len(names):
        raise ValueError(f"arms {names} name an arm more than once")
    unknown = [n for n in names if n not in spec.arms]
    if unknown:
        raise ValueError(f"unknown arms {unknown}; the protocol has {sorted(spec.arms)}")
    return [spec.arms[n] for n in names]


def open_backends(spec: Spec, mnemo: MnemoClient, env: Mapping[str, str] = os.environ) -> Backends:
    """mnemo with the reader and judge of the protocol; the fakes when it names "fake"."""
    return Backends(
        mnemo=mnemo,
        reader=models.open_model(spec.reader.model, spec.reader.env, models.FakeReader(), env),
        judge=models.open_model(spec.judge.model, spec.judge.env, models.FakeJudge(), env),
    )


def load_questions(spec: Spec, split: str | None) -> list[Question]:
    """Returns the protocol's questions, only those of split when one is given.

    A protocol names either a toy dataset file, with its splits as lists of
    ids, or a LongMemEval file of its datasets, split with its seed.
    """
    if "dataset" in spec.cfg:
        return _toy_questions(spec, split)
    data = longmemeval.load_data_protocol(spec.path)
    f = data.files[spec.cfg["questions"]]
    if not f.path.exists() or longmemeval.file_sha256(f.path) != f.sha256:
        raise ValueError(f"{f.path} is missing or not the pinned file; run mnemo-eval fetch")
    questions = longmemeval.load_longmemeval(f.path)
    return questions if split is None else longmemeval.split(questions, split, data.seed)


def _toy_questions(spec: Spec, split: str | None) -> list[Question]:
    path = spec.path.parent / spec.cfg["dataset"]
    questions = [
        _toy_question(q) for q in json.loads(path.read_text(encoding="utf-8"))["questions"]
    ]
    if split is None:
        return questions
    splits = spec.cfg.get("splits", {})
    if split not in splits:
        raise ValueError(f"unknown split {split!r}; the protocol has {sorted(splits)}")
    wanted = set(splits[split])
    return [q for q in questions if q.id in wanted]


def _toy_question(q: dict[str, Any]) -> Question:
    sessions = [
        Session(id=s["session_id"], date=s["date"], messages=s["messages"], evidence_turns=[])
        for s in q["sessions"]
    ]
    return Question(
        id=q["question_id"],
        type=q["type"],
        question=q["question"],
        answer=q["answer"],
        question_date=q["question_date"],
        sessions=sessions,
        evidence_session_ids=[],
        is_abstention=False,
    )


def ingest(client: MnemoClient, subject: str, question: Question) -> None:
    """Sends each session of the history, in date order, as one request.

    A session id that repeats within the history gets a #dup<n> suffix, which
    the recall arm strips. Empty messages are left out and longer ones split,
    as the service refuses them.
    """
    seen: Counter[str] = Counter()
    for session in sorted(question.sessions, key=lambda s: arms.parse_date(s.date)):
        occurred_at = arms.parse_date(session.date).strftime("%Y-%m-%dT%H:%M:%SZ")
        messages = [
            {"role": m["role"], "content": part, "occurred_at": occurred_at}
            for m in session.messages
            for part in _parts(m["content"])
        ]
        seen[session.id] += 1
        if not messages:
            continue
        session_id = (
            session.id if seen[session.id] == 1 else f"{session.id}#dup{seen[session.id] - 1}"
        )
        client.remember(subject, session_id, messages, wait=True)


def _parts(content: str) -> list[str]:
    """Splits content into pieces of at most MAX_CONTENT_BYTES of UTF-8."""
    parts = []
    data = content.encode()
    while data:
        cut = min(len(data), MAX_CONTENT_BYTES)
        # Back off to the start of a character.
        while cut < len(data) and data[cut] & UTF8_CONTINUATION_MASK == UTF8_CONTINUATION:
            cut -= 1
        parts.append(data[:cut].decode())
        data = data[cut:]
    return parts


def answer(
    spec: Spec, arm: arms.ArmConfig, question: Question, subject: str, backends: Backends
) -> dict[str, Any]:
    """Runs one arm on one question, reads and judges; returns the result row."""
    out = arms.ARMS[arm.memory](question, arm, backends.mnemo, subject)
    prompt = spec.reader_prompt.format(
        memory=out.context, date=arms.show_date(question.question_date), question=question.question
    )
    reply = backends.reader.complete(prompt)
    return {
        "protocol": spec.id,
        "question_id": question.id,
        "type": question.type,
        "is_abstention": question.is_abstention,
        "answer": reply.text,
        "correct": judge.judge(question, reply.text, backends.judge),
        "reader_input_tokens": reply.input_tokens,
        "reader_output_tokens": reply.output_tokens,
        "context_tokens": out.context_tokens,
        "recall_ms": out.recall_ms,
        "retrieved_session_ids": out.retrieved_session_ids,
        "evidence_session_ids": question.evidence_session_ids,
    }


def run(
    spec: Spec,
    arm_cfgs: list[arms.ArmConfig],
    questions: list[Question],
    backends: Backends,
    runs_dir: Path,
) -> dict[str, Path]:
    """Runs every arm over questions into runs_dir/<protocol>/<arm>.jsonl.

    Rows are appended as each question ends, so a run that stops keeps what it
    did. Each run stores the history under fresh subjects, so repeating it never
    recalls episodes of an earlier run.
    """
    run_id = secrets.token_hex(4)
    out_dir = runs_dir / spec.id
    out_dir.mkdir(parents=True, exist_ok=True)
    paths = {a.name: out_dir / f"{a.name}.jsonl" for a in arm_cfgs}
    for path in paths.values():
        path.write_bytes(b"")
    needs_memory = any(a.memory == "recall" for a in arm_cfgs)
    for q in questions:
        subject = f"eval-{run_id}-{q.id}"
        if needs_memory:
            ingest(backends.mnemo, subject, q)
        for arm in arm_cfgs:
            append(paths[arm.name], answer(spec, arm, q, subject, backends))
    return paths


def append(path: Path, row: dict[str, Any]) -> None:
    """Appends row as one JSON line."""
    with path.open("a", encoding="utf-8", newline="\n") as f:
        f.write(json.dumps(row, ensure_ascii=False) + "\n")


def read_rows(path: Path) -> list[dict[str, Any]]:
    return [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line]
