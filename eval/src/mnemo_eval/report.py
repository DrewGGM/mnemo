"""The report of a protocol's runs: summary.json and its table in Markdown,
and the check that a results directory never mixes protocols."""

import json
from pathlib import Path
from typing import Any

from mnemo_eval import metrics
from mnemo_eval.pipeline import Spec, read_rows

BOOTSTRAP_RESAMPLES = 10_000
SUMMARY = "summary.json"
TABLE = "results.md"
PER_MILLION = 1_000_000


def load_runs(spec: Spec, runs_dir: Path) -> dict[str, list[dict[str, Any]]]:
    """The rows of each arm of spec that has a results file, in protocol order.

    Every arm must answer the same questions once each, so that the table
    compares arms on one question set: runs of other splits are refused.
    """
    out_dir = runs_dir / spec.id
    paths = {name: out_dir / f"{name}.jsonl" for name in spec.arms}
    runs = {name: read_rows(p) for name, p in paths.items() if p.exists()}
    runs = {name: rows for name, rows in runs.items() if rows}
    if not runs:
        raise ValueError(f"no results of protocol {spec.id} under {runs_dir}")
    asked = {name: [r["question_id"] for r in rows] for name, rows in runs.items()}
    if any(len(set(ids)) != len(ids) for ids in asked.values()):
        raise ValueError(f"results under {out_dir} answer a question more than once")
    if len({frozenset(ids) for ids in asked.values()}) > 1:
        raise ValueError(f"arms under {out_dir} answer different questions; rerun them together")
    return runs


def summarize(spec: Spec, runs: dict[str, list[dict[str, Any]]]) -> dict[str, Any]:
    """Accuracy, retrieval, tokens, latency and reader cost of each arm."""
    return {
        "protocol": spec.id,
        "reader": spec.reader.model,
        "judge": spec.judge.model,
        "prices": spec.prices,
        "bootstrap": {"resamples": BOOTSTRAP_RESAMPLES, "seed": spec.seed, "level": 0.95},
        "arms": {name: _arm_summary(spec, name, rows) for name, rows in runs.items()},
    }


def _arm_summary(spec: Spec, name: str, rows: list[dict[str, Any]]) -> dict[str, Any]:
    correct = [float(r["correct"]) for r in rows]
    abstention = [r for r in rows if r["is_abstention"]]
    recalled = spec.arms[name].memory == "recall"
    recall_ms = [r["recall_ms"] for r in rows if r["recall_ms"] is not None]
    return {
        "questions": len(rows),
        "accuracy": metrics.accuracy(rows),
        "accuracy_ci95": metrics.bootstrap_ci(correct, spec.seed, BOOTSTRAP_RESAMPLES),
        "accuracy_by_type": metrics.accuracy_by_type(rows),
        "abstention_questions": len(abstention),
        "abstention_accuracy": metrics.accuracy(abstention) if abstention else None,
        "recall_at_5": metrics.recall_at_k(rows, 5) if recalled else None,
        "recall_at_10": metrics.recall_at_k(rows, 10) if recalled else None,
        "mrr": metrics.mrr(rows) if recalled else None,
        "reader_input_tokens": _spread([r["reader_input_tokens"] for r in rows]),
        "context_tokens": _spread([r["context_tokens"] for r in rows]),
        "recall_ms": _latency(recall_ms) if recall_ms else None,
        "reader_cost_usd": _cost(spec, rows),
    }


def _spread(values: list[float]) -> dict[str, float]:
    return {"mean": sum(values) / len(values), "p95": metrics.percentile(values, 95)}


def _latency(values: list[float]) -> dict[str, float]:
    return {f"p{p}": metrics.percentile(values, p) for p in (50, 95, 99)}


def _cost(spec: Spec, rows: list[dict[str, Any]]) -> dict[str, float] | None:
    """Reader cost at the dated price list; None when the reader has no price."""
    price = spec.prices.get("usd_per_million_tokens", {}).get(spec.reader.model)
    if price is None:
        return None
    total = sum(
        r["reader_input_tokens"] * price["input"] + r["reader_output_tokens"] * price["output"]
        for r in rows
    )
    total /= PER_MILLION
    return {"total": total, "per_question": total / len(rows)}


def _pct(value: float | None) -> str:
    return "n/a" if value is None else f"{value * 100:.1f}%"


def _usd(cost: dict[str, float] | None) -> str:
    return "n/a" if cost is None else f"{cost['total']:.4f} / {cost['per_question']:.6f}"


def _ms(latency: dict[str, float] | None) -> str:
    return "n/a" if latency is None else f"{latency['p50']:.0f} / {latency['p95']:.0f}"


def _row(name: str, arm: dict[str, Any], types: list[str]) -> str:
    low, high = arm["accuracy_ci95"]
    cells = [
        name,
        str(arm["questions"]),
        f"{_pct(arm['accuracy'])} ({_pct(low)}-{_pct(high)})",
        *(_pct(arm["accuracy_by_type"].get(t)) for t in types),
        _pct(arm["abstention_accuracy"]),
        f"{arm['reader_input_tokens']['mean']:.0f} / {arm['reader_input_tokens']['p95']:.0f}",
        _usd(arm["reader_cost_usd"]),
        _ms(arm["recall_ms"]),
        f"{_pct(arm['recall_at_5'])} / {_pct(arm['recall_at_10'])}",
        "n/a" if arm["mrr"] is None else f"{arm['mrr']:.3f}",
    ]
    return "| " + " | ".join(cells) + " |"


def to_markdown(summary: dict[str, Any]) -> str:
    """The results table, followed by the models and the dated price list."""
    arms = summary["arms"]
    types = sorted({t for arm in arms.values() for t in arm["accuracy_by_type"]})
    header = [
        "Arm",
        "Questions",
        "Accuracy (95% CI)",
        *types,
        "Abstention",
        "Reader input tokens (mean / p95)",
        "Reader cost USD (total / per question)",
        "recall ms (p50 / p95)",
        "Session recall@5 / @10",
        "MRR",
    ]
    lines = [
        f"# Results of protocol {summary['protocol']}",
        "",
        "| " + " | ".join(header) + " |",
        "|" + "---|" * len(header),
        *(_row(name, arm, types) for name, arm in arms.items()),
        "",
        f"Reader: {summary['reader']}. Judge: {summary['judge']}, with the official "
        "LongMemEval prompts. Intervals: percentile bootstrap over questions, "
        f"{summary['bootstrap']['resamples']} resamples, seed {summary['bootstrap']['seed']}.",
        "",
        *_price_lines(summary["prices"]),
    ]
    return "\n".join(lines) + "\n"


def _price_lines(prices: dict[str, Any]) -> list[str]:
    table = prices.get("usd_per_million_tokens", {})
    if not table:
        return ["Reader cost: no price list in the protocol."]
    lines = [
        f"Reader cost = input tokens x input price + output tokens x output price, "
        f"at the prices of {prices.get('date')} (USD per million tokens; source: "
        f"{prices.get('source')}):",
        "",
    ]
    lines += [f"- {model}: input {p['input']}, output {p['output']}" for model, p in table.items()]
    return lines


def write(summary: dict[str, Any], out: Path) -> tuple[Path, Path]:
    """Writes out/<protocol>/summary.json and results.md."""
    out_dir = out / summary["protocol"]
    out_dir.mkdir(parents=True, exist_ok=True)
    json_path, md_path = out_dir / SUMMARY, out_dir / TABLE
    json_path.write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8", newline="\n")
    md_path.write_text(to_markdown(summary), encoding="utf-8", newline="\n")
    return json_path, md_path


def verify_protocol(root: Path) -> tuple[int, list[str]]:
    """Checks every directory under root holding results: one protocol id each.

    Returns how many directories hold results and the problems found.
    """
    if not root.is_dir():
        raise ValueError(f"{root} is not a directory")
    checked, problems = 0, []
    for directory in [root, *sorted(p for p in root.rglob("*") if p.is_dir())]:
        ids = _protocol_ids(directory, problems)
        if ids is None:
            continue
        checked += 1
        if len(ids) > 1:
            problems.append(f"{directory}: results of {len(ids)} protocols: {sorted(ids)}")
    return checked, problems


def _protocol_ids(directory: Path, problems: list[str]) -> set[str] | None:
    """The protocol ids of the results files in directory; None if it has none."""
    files = sorted(directory.glob("*.jsonl"))
    summary = directory / SUMMARY
    if not files and not summary.exists():
        return None
    found = [(f"{path}:{n}", row) for path in files for n, row in enumerate(read_rows(path), 1)]
    if summary.exists():
        found.append((str(summary), json.loads(summary.read_text(encoding="utf-8"))))
    ids = set()
    for where, record in found:
        protocol = record.get("protocol")
        if protocol:
            ids.add(protocol)
        else:
            problems.append(f"{where}: no protocol id")
    return ids
