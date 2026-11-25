"""Command line of the evaluation harness: mnemo-eval."""

import argparse
import json
import sys
from pathlib import Path

import yaml

from mnemo_eval import pipeline
from mnemo_eval.client import MnemoClient, MnemoError
from mnemo_eval.datasets import longmemeval

# The mnemo service of docker-compose.yml.
BASE_URL = "http://127.0.0.1:8080"
# Commands run from the repository root (CONTRIBUTING.md).
DEFAULT_PROTOCOL = Path("eval/config/protocol.yaml")


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="mnemo-eval")
    commands = parser.add_subparsers(dest="command", required=True)
    run = commands.add_parser("run", help="run arms of a protocol and write their results")
    run.add_argument("--protocol", required=True, type=Path, help="protocol YAML file")
    run.add_argument("--arms", required=True, help='comma separated arm names, or "all"')
    run.add_argument("--split", help="only the questions of this split of the protocol")
    fetch = commands.add_parser("fetch", help="download the datasets and check their SHA-256")
    fetch.add_argument("--protocol", default=DEFAULT_PROTOCOL, type=Path, help="protocol YAML file")
    args = parser.parse_args(argv)
    try:
        if args.command == "fetch":
            return _fetch(args.protocol)
        return _run(args.protocol, args.arms.split(","), args.split)
    except (MnemoError, ValueError, KeyError, OSError, yaml.YAMLError) as err:
        print(f"mnemo-eval: {type(err).__name__}: {err}", file=sys.stderr)
        return 1


def _run(protocol: Path, arm_names: list[str], split: str | None) -> int:
    spec = pipeline.load_protocol(protocol)
    arms = pipeline.select_arms(spec, arm_names)
    questions = pipeline.load_questions(spec, split)
    # Protocols live in eval/config; runs go to eval/runs.
    runs_dir = protocol.resolve().parent.parent / "runs"
    paths = pipeline.run(spec, arms, questions, MnemoClient(BASE_URL), runs_dir)
    print(f"protocol {spec.id}")
    print(f"{'arm':<6} {'correct':>9}  results")
    for name, path in paths.items():
        rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
        correct = sum(r["correct"] for r in rows)
        print(f"{name:<6} {correct:>4}/{len(rows):<4}  {path}")
    return 0


def _fetch(protocol: Path) -> int:
    data = longmemeval.load_data_protocol(protocol)
    unpinned = []
    for name, f in data.files.items():
        got = longmemeval.download(f)
        print(f"{name}: {f.path} sha256 {got}")
        if not f.sha256:
            unpinned.append(name)
    if unpinned:
        print(f"mnemo-eval: pin the sha256 of {', '.join(unpinned)} in {protocol}", file=sys.stderr)
        return 1
    _describe(longmemeval.load_longmemeval(data.files["longmemeval_s"].path), data.seed)
    return 0


def _describe(questions: list[longmemeval.Question], seed: int) -> None:
    abstention = sum(q.is_abstention for q in questions)
    print(f"longmemeval_s: {len(questions)} questions, {abstention} abstention")
    print(f"{'type':<26} {'questions':>9} {'abstention':>10} {'dev60':>5}")
    dev = longmemeval.split(questions, "dev60", seed)
    for qtype in sorted({q.type for q in questions}):
        of_type = [q for q in questions if q.type == qtype]
        abs_count = sum(q.is_abstention for q in of_type)
        dev_count = sum(q.type == qtype for q in dev)
        print(f"{qtype:<26} {len(of_type):>9} {abs_count:>10} {dev_count:>5}")
    sizes = {name: len(longmemeval.split(questions, name, seed)) for name in longmemeval.SPLITS}
    print("splits: " + ", ".join(f"{name} {n}" for name, n in sizes.items()))
    rep = longmemeval.session_repetition(questions)
    print(
        f"sessions whose content repeats in another question (A-6): {rep.repeated} of "
        f"{rep.sessions} ({rep.repeated / rep.sessions:.1%}); {rep.distinct} distinct, so a "
        f"cache by content skips {1 - rep.distinct / rep.sessions:.1%} of the ingestions"
    )
