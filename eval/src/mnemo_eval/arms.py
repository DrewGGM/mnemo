"""The arms of docs/evaluation-plan.md §3: the memory each one hands the reader.

B0 gives none, B1 the whole history, and B2 and M what mnemo recalls with the
parameters of the protocol. Every memory line has the same shape, so that the
arms differ only in what they choose.
"""

import re
import time
from collections.abc import Callable
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any

from mnemo_eval.client import MnemoClient
from mnemo_eval.datasets.longmemeval import Question

MEMORY_KINDS = ("none", "history", "recall")
# LongMemEval dates, such as "2023/05/20 (Sat) 02:21", carry no zone.
LONGMEMEVAL_DATE = "%Y/%m/%d (%a) %H:%M"
SHOWN_DATE = "%Y-%m-%d (%a) %H:%M"
# A session id repeated within one history is stored as <id>#dup<n>.
DUPLICATE_SUFFIX = re.compile(r"#dup\d+$")


@dataclass(frozen=True)
class ArmConfig:
    name: str
    memory: str
    recall: dict[str, Any]


@dataclass(frozen=True)
class ArmOutput:
    context: str
    context_tokens: int
    recall_ms: float | None
    retrieved_session_ids: list[str]


def parse_date(text: str) -> datetime:
    """Reads a LongMemEval date, or an RFC 3339 one, as UTC."""
    try:
        parsed = datetime.strptime(text, LONGMEMEVAL_DATE)
    except ValueError:
        parsed = datetime.fromisoformat(text)
    return parsed.replace(tzinfo=UTC) if parsed.tzinfo is None else parsed.astimezone(UTC)


def show_date(text: str) -> str:
    return parse_date(text).strftime(SHOWN_DATE)


def memory_line(date: str, role: str, content: str) -> str:
    return f"[{show_date(date)}] {role}: {content}"


def estimate_tokens(text: str) -> int:
    """Characters / 4, rounded up, as the service estimates them."""
    return (len(text) + 3) // 4


def _output(lines: list[str], recall_ms: float | None, session_ids: list[str]) -> ArmOutput:
    context = "\n".join(lines)
    return ArmOutput(context, estimate_tokens(context), recall_ms, session_ids)


def no_memory(question: Question, cfg: ArmConfig, client: MnemoClient, subject: str) -> ArmOutput:
    """B0: the reader gets only the question and its date."""
    return _output([], None, [])


def full_history(
    question: Question, cfg: ArmConfig, client: MnemoClient, subject: str
) -> ArmOutput:
    """B1: every message of the history, in chronological order."""
    sessions = sorted(question.sessions, key=lambda s: parse_date(s.date))
    lines = [memory_line(s.date, m["role"], m["content"]) for s in sessions for m in s.messages]
    return _output(lines, None, [])


def recall(question: Question, cfg: ArmConfig, client: MnemoClient, subject: str) -> ArmOutput:
    """B2 and M: what mnemo recalls for the question, in the order it ranks it."""
    start = time.perf_counter()
    episodes = client.recall(subject, question.question, **cfg.recall)["episodes"]
    elapsed_ms = (time.perf_counter() - start) * 1000
    lines = [memory_line(e["occurred_at"], e["role"], e["content"]) for e in episodes]
    session_ids = [DUPLICATE_SUFFIX.sub("", e["session_id"]) for e in episodes]
    return _output(lines, elapsed_ms, session_ids)


ARMS: dict[str, Callable[[Question, ArmConfig, MnemoClient, str], ArmOutput]] = {
    "none": no_memory,
    "history": full_history,
    "recall": recall,
}
