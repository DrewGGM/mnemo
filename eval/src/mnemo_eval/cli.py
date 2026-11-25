"""Command line of the evaluation harness: mnemo-eval."""

import argparse
import sys
from pathlib import Path

import yaml

from mnemo_eval import pipeline, report
from mnemo_eval.client import MnemoClient, MnemoError
from mnemo_eval.datasets import longmemeval
from mnemo_eval.models import ModelError

# The mnemo service of docker-compose.yml.
BASE_URL = "http://127.0.0.1:8080"
# Commands run from the repository root (CONTRIBUTING.md).
DEFAULT_PROTOCOL = Path("eval/config/protocol.yaml")
ERRORS = (MnemoError, ModelError, ValueError, KeyError, OSError, yaml.YAMLError)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="mnemo-eval")
    commands = parser.add_subparsers(dest="command", required=True)
    run = commands.add_parser("run", help="run arms of a protocol and write their results")
    run.add_argument("--protocol", required=True, type=Path, help="protocol YAML file")
    run.add_argument("--arms", required=True, help='comma separated arm names, or "all"')
    run.add_argument("--split", help="only the questions of this split of the protocol")
    fetch = commands.add_parser("fetch", help="download the datasets and check their SHA-256")
    fetch.add_argument("--protocol", default=DEFAULT_PROTOCOL, type=Path, help="protocol YAML file")
    rep = commands.add_parser("report", help="write summary.json and the results table")
    rep.add_argument("--out", required=True, type=Path, help="directory of the report")
    rep.add_argument("--protocol", default=DEFAULT_PROTOCOL, type=Path, help="protocol YAML file")
    verify = commands.add_parser(
        "verify-protocol", help="fail if a results directory mixes protocol ids"
    )
    verify.add_argument("dir", type=Path, help="directory of runs or results")
    args = parser.parse_args(argv)
    try:
        return _dispatch(args)
    except ERRORS as err:
        print(f"mnemo-eval: {type(err).__name__}: {err}", file=sys.stderr)
        return 1


def _dispatch(args: argparse.Namespace) -> int:
    if args.command == "fetch":
        return _fetch(args.protocol)
    if args.command == "report":
        return _report(args.protocol, args.out)
    if args.command == "verify-protocol":
        return _verify(args.dir)
    return _run(args.protocol, args.arms.split(","), args.split)


def _run(protocol: Path, arm_names: list[str], split: str | None) -> int:
    spec = pipeline.load_protocol(protocol)
    arms = pipeline.select_arms(spec, arm_names)
    questions = pipeline.load_questions(spec, split)
    backends = pipeline.open_backends(spec, MnemoClient(BASE_URL))
    paths = pipeline.run(spec, arms, questions, backends, _runs_dir(protocol))
    print(f"protocol {spec.id}")
    print(f"{'arm':<6} {'correct':>9}  results")
    for name, path in paths.items():
        rows = pipeline.read_rows(path)
        correct = sum(r["correct"] for r in rows)
        print(f"{name:<6} {correct:>4}/{len(rows):<4}  {path}")
    return 0


def _runs_dir(protocol: Path) -> Path:
    # Protocols live in eval/config; runs go to eval/runs.
    return protocol.resolve().parent.parent / "runs"


def _report(protocol: Path, out: Path) -> int:
    spec = pipeline.load_protocol(protocol)
    summary = report.summarize(spec, report.load_runs(spec, _runs_dir(protocol)))
    for path in report.write(summary, out):
        print(path)
    return 0


def _verify(directory: Path) -> int:
    checked, problems = report.verify_protocol(directory)
    for problem in problems:
        print(f"mnemo-eval: {problem}", file=sys.stderr)
    print(f"{checked} result directories under {directory}, {len(problems)} problems")
    return 1 if problems else 0


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
