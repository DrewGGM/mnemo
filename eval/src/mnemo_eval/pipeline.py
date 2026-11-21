"""The evaluation pipeline, one function per stage: ingest, query, read, judge, write.

Only the fake reader and judge exist; the real ones arrive with T-009.
"""

import hashlib
import json
import re
import secrets
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml

from mnemo_eval.client import MnemoClient

FAKE = "fake"
NO_ANSWER = "I don't know"
MEMORY_KINDS = ("none", "recall")
# Arm names become file names under eval/runs.
ARM_NAME = re.compile(r"[A-Za-z0-9_-]+")

READER_INSTRUCTIONS = (
    "Answer the question using only the memory below. "
    f'If the memory does not contain the answer, reply "{NO_ANSWER}".'
)


@dataclass(frozen=True)
class Arm:
    name: str
    memory: str
    recall: dict[str, Any]


@dataclass(frozen=True)
class Spec:
    """A loaded protocol file; id is the SHA-256 of its bytes."""

    id: str
    dataset: Path
    arms: dict[str, Arm]
    splits: dict[str, list[str]]


def load_protocol(path: Path) -> Spec:
    """Reads a protocol YAML; the dataset path is relative to the YAML file."""
    raw = path.read_bytes()
    cfg = yaml.safe_load(raw)
    if not isinstance(cfg, dict) or not isinstance(cfg.get("arms"), dict):
        raise ValueError(f"{path}: expected a mapping with an arms mapping")
    for role in ("reader", "judge"):
        if cfg.get(role) != FAKE:
            raise ValueError(f"{path}: {role} must be {FAKE!r}; real models arrive with T-009")
    arms = {name: _arm(name, a) for name, a in cfg["arms"].items()}
    return Spec(
        id=hashlib.sha256(raw).hexdigest(),
        dataset=path.parent / cfg["dataset"],
        arms=arms,
        splits=cfg.get("splits", {}),
    )


def _arm(name: str, cfg: dict[str, Any]) -> Arm:
    if not ARM_NAME.fullmatch(str(name)):
        raise ValueError(f"arm name {name!r} must use only letters, digits, - and _")
    memory = cfg["memory"]
    if memory not in MEMORY_KINDS:
        raise ValueError(f"arm {name}: memory must be one of {MEMORY_KINDS}, got {memory!r}")
    return Arm(name=name, memory=memory, recall=cfg.get("recall", {}))


def select_arms(spec: Spec, names: list[str]) -> list[Arm]:
    """Returns the arms named, or every arm for ["all"]."""
    if names == ["all"]:
        return list(spec.arms.values())
    unknown = [n for n in names if n not in spec.arms]
    if unknown:
        raise ValueError(f"unknown arms {unknown}; the protocol has {sorted(spec.arms)}")
    return [spec.arms[n] for n in names]


def load_questions(spec: Spec, split: str | None) -> list[dict[str, Any]]:
    """Returns the dataset questions, only those of split when one is given."""
    questions = json.loads(spec.dataset.read_text(encoding="utf-8"))["questions"]
    if split is None:
        return questions
    if split not in spec.splits:
        raise ValueError(f"unknown split {split!r}; the protocol has {sorted(spec.splits)}")
    wanted = set(spec.splits[split])
    return [q for q in questions if q["question_id"] in wanted]


def ingest(client: MnemoClient, subject: str, question: dict[str, Any]) -> None:
    """Sends each session of the history, in date order, as one request."""
    for session in sorted(question["sessions"], key=lambda s: s["date"]):
        messages = [{**m, "occurred_at": session["date"]} for m in session["messages"]]
        client.remember(subject, session["session_id"], messages, wait=True)


def query(
    client: MnemoClient, subject: str, question: dict[str, Any], params: dict[str, Any]
) -> tuple[list[str], float]:
    """Recalls for the question; returns the episode contents and the wall time in ms."""
    start = time.perf_counter()
    res = client.recall(subject, question["question"], **params)
    elapsed_ms = (time.perf_counter() - start) * 1000
    return [e["content"] for e in res["episodes"]], elapsed_ms


def read(question: dict[str, Any], context: list[str]) -> tuple[str, int]:
    """The fake reader: answers with the first memory line, or NO_ANSWER without memory.

    Returns the answer and the input tokens of the prompt a real reader would get,
    estimated as characters / 4 like the service does.
    """
    prompt = "\n\n".join(
        [
            READER_INSTRUCTIONS,
            "Memory:\n" + "\n".join(context),
            f"Current date: {question['question_date']}",
            f"Question: {question['question']}",
        ]
    )
    answer = context[0] if context else NO_ANSWER
    return answer, (len(prompt) + 3) // 4


def judge(question: dict[str, Any], answer: str) -> bool:
    """The fake judge: correct when the gold answer appears in the answer."""
    return question["answer"].casefold() in answer.casefold()


def run(
    spec: Spec,
    arms: list[Arm],
    questions: list[dict[str, Any]],
    client: MnemoClient,
    runs_dir: Path,
) -> dict[str, Path]:
    """Runs every arm over questions and writes runs_dir/<protocol>/<arm>.jsonl.

    Each run stores the history under fresh subjects, so repeating it never
    recalls episodes of an earlier run.
    """
    run_id = secrets.token_hex(4)
    subjects = {q["question_id"]: f"eval-{run_id}-{q['question_id']}" for q in questions}
    if any(a.memory == "recall" for a in arms):
        for q in questions:
            ingest(client, subjects[q["question_id"]], q)
    out_dir = runs_dir / spec.id
    out_dir.mkdir(parents=True, exist_ok=True)
    paths = {}
    for arm in arms:
        rows = [_answer(spec, arm, q, client, subjects[q["question_id"]]) for q in questions]
        paths[arm.name] = write(out_dir / f"{arm.name}.jsonl", rows)
    return paths


def _answer(
    spec: Spec, arm: Arm, question: dict[str, Any], client: MnemoClient, subject: str
) -> dict[str, Any]:
    context: list[str] = []
    recall_ms = None
    if arm.memory == "recall":
        context, recall_ms = query(client, subject, question, arm.recall)
    answer, reader_tokens = read(question, context)
    return {
        "protocol": spec.id,
        "question_id": question["question_id"],
        "type": question["type"],
        "answer": answer,
        "correct": judge(question, answer),
        "reader_input_tokens": reader_tokens,
        "recall_ms": recall_ms,
    }


def write(path: Path, rows: list[dict[str, Any]]) -> Path:
    """Writes one JSON line per row."""
    with path.open("w", encoding="utf-8", newline="\n") as f:
        for row in rows:
            f.write(json.dumps(row, ensure_ascii=False) + "\n")
    return path
