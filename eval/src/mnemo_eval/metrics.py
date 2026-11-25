"""Metrics of docs/evaluation-plan.md §5, over the result rows of one arm.

A row is one line of eval/runs/<protocol>/<arm>.jsonl. Retrieval metrics follow
the official LongMemEval scripts: session level, abstention questions left
out, and recall@k counts a question only when every evidence session is among
the first k distinct sessions retrieved.
"""

import math
import random
from collections import defaultdict
from collections.abc import Sequence
from typing import Any

Row = dict[str, Any]
HUNDRED = 100


def accuracy(rows: Sequence[Row]) -> float:
    """Fraction of rows the judge marked correct."""
    if not rows:
        raise ValueError("accuracy of no rows")
    return sum(bool(r["correct"]) for r in rows) / len(rows)


def accuracy_by_type(rows: Sequence[Row]) -> dict[str, float]:
    """Accuracy of each question type, by type name."""
    by_type: dict[str, list[Row]] = defaultdict(list)
    for r in rows:
        by_type[r["type"]].append(r)
    return {t: accuracy(by_type[t]) for t in sorted(by_type)}


def _ranked_sessions(row: Row) -> list[str]:
    return list(dict.fromkeys(row["retrieved_session_ids"]))


def _with_evidence(rows: Sequence[Row]) -> list[Row]:
    return [r for r in rows if not r["is_abstention"] and r["evidence_session_ids"]]


def recall_at_k(rows: Sequence[Row], k: int) -> float | None:
    """Fraction of answerable questions whose evidence sessions are all in the top k.

    None when no row is answerable with evidence.
    """
    scored = _with_evidence(rows)
    if not scored:
        return None
    hits = sum(set(r["evidence_session_ids"]) <= set(_ranked_sessions(r)[:k]) for r in scored)
    return hits / len(scored)


def mrr(rows: Sequence[Row]) -> float | None:
    """Mean reciprocal rank of the first evidence session; 0 when none is retrieved."""
    scored = _with_evidence(rows)
    if not scored:
        return None
    total = 0.0
    for r in scored:
        evidence = set(r["evidence_session_ids"])
        ranks = [i for i, sid in enumerate(_ranked_sessions(r), 1) if sid in evidence]
        total += 1 / ranks[0] if ranks else 0.0
    return total / len(scored)


def percentile(values: Sequence[float], p: float) -> float:
    """The p-th percentile of the whole list, interpolating linearly between ranks."""
    if not values:
        raise ValueError("percentile of no values")
    if not 0 <= p <= HUNDRED:
        raise ValueError(f"percentile {p} is outside [0, 100]")
    ordered = sorted(values)
    rank = p / HUNDRED * (len(ordered) - 1)
    low = math.floor(rank)
    high = min(low + 1, len(ordered) - 1)
    return ordered[low] + (ordered[high] - ordered[low]) * (rank - low)


def bootstrap_ci(
    values: Sequence[float], seed: int, resamples: int = 10_000, level: float = 0.95
) -> tuple[float, float]:
    """Percentile bootstrap interval of the mean, resampling values with a seeded generator."""
    if not values:
        raise ValueError("bootstrap of no values")
    rng = random.Random(seed)  # noqa: S311 - resampling needs a seed, not secrecy
    n = len(values)
    means = [sum(rng.choices(values, k=n)) / n for _ in range(resamples)]
    tail = (1 - level) / 2 * HUNDRED
    return percentile(means, tail), percentile(means, HUNDRED - tail)
