"""Command line of the evaluation harness: mnemo-eval."""

import argparse
import json
import sys
from pathlib import Path

import yaml

from mnemo_eval import pipeline
from mnemo_eval.client import MnemoClient, MnemoError

# The mnemo service of docker-compose.yml.
BASE_URL = "http://127.0.0.1:8080"


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="mnemo-eval")
    commands = parser.add_subparsers(dest="command", required=True)
    run = commands.add_parser("run", help="run arms of a protocol and write their results")
    run.add_argument("--protocol", required=True, type=Path, help="protocol YAML file")
    run.add_argument("--arms", required=True, help='comma separated arm names, or "all"')
    run.add_argument("--split", help="only the questions of this split of the protocol")
    args = parser.parse_args(argv)
    try:
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
