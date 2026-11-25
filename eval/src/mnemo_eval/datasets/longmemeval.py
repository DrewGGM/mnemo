"""LongMemEval_S: download with a pinned SHA-256, load into Questions, and split.

Format and host: https://github.com/xiaowu0162/LongMemEval (README, "Dataset Format").
"""

import hashlib
import json
import os
import tempfile
import urllib.request
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Any, BinaryIO
from urllib.parse import urlparse

import yaml

ABSTENTION_SUFFIX = "_abs"
# dev60 and write100 are stratified samples; heldout440 is what dev60 leaves.
SAMPLE_SIZES = {"dev60": 60, "write100": 100}
SPLITS = ("dev60", "heldout440", "write100", "all")
QUESTION_FIELDS = (
    "question_id",
    "question_type",
    "question",
    "answer",
    "question_date",
    "haystack_session_ids",
    "haystack_dates",
    "haystack_sessions",
    "answer_session_ids",
)
TURN_FIELDS = ("role", "content")
DOWNLOAD_TIMEOUT_S = 60
CHUNK_BYTES = 1 << 20
# The real files are a few hundred MB.
MAX_DOWNLOAD_BYTES = 2 << 30


@dataclass(frozen=True)
class Session:
    """A history session; messages carry only role and content, so that the
    has_answer labels, kept as evidence_turns indices, never reach the system."""

    id: str
    date: str
    messages: list[dict[str, str]]
    evidence_turns: list[int]


@dataclass(frozen=True)
class Question:
    id: str
    type: str
    question: str
    answer: str
    question_date: str
    sessions: list[Session]
    evidence_session_ids: list[str]
    is_abstention: bool


@dataclass(frozen=True)
class DataFile:
    url: str
    sha256: str
    path: Path


@dataclass(frozen=True)
class DataProtocol:
    """The data part of a protocol file: the split seed and the files to fetch."""

    seed: int
    files: dict[str, DataFile]


@dataclass(frozen=True)
class Repetition:
    """How often a session's content appears in the histories of several questions."""

    sessions: int
    repeated: int
    distinct: int


def load_data_protocol(path: Path) -> DataProtocol:
    """Reads seed, data_dir and datasets of a protocol YAML; data_dir is relative to it."""
    cfg = yaml.safe_load(path.read_bytes())
    data_dir = (path.parent / cfg["data_dir"]).resolve()
    files = {}
    for name, spec in cfg["datasets"].items():
        url = spec["url"]
        if urlparse(url).scheme != "https":
            raise ValueError(f"{path}: dataset {name} must be fetched over https, got {url!r}")
        file_name = Path(urlparse(url).path).name
        files[name] = DataFile(url=url, sha256=spec.get("sha256") or "", path=data_dir / file_name)
    return DataProtocol(seed=int(cfg["seed"]), files=files)


def download(f: DataFile) -> str:
    """Downloads f unless a copy with its pinned SHA-256 is already there.

    Returns the SHA-256 of the file. A download whose SHA-256 differs from the
    pinned one raises ValueError and leaves nothing behind; without a pinned
    SHA-256 the file is kept so that its printed SHA-256 can be pinned.
    """
    if f.sha256 and f.path.exists() and file_sha256(f.path) == f.sha256:
        return f.sha256
    f.path.parent.mkdir(parents=True, exist_ok=True)
    # A name of its own, so that two fetches at once do not write the same file.
    fd, name = tempfile.mkstemp(dir=f.path.parent, prefix=f.path.name + ".", suffix=".part")
    part = Path(name)
    try:
        with os.fdopen(fd, "wb") as out:
            got = _fetch_into(f.url, out)
        if f.sha256 and got != f.sha256:
            raise ValueError(f"{f.url}: SHA-256 {got} does not match the protocol's {f.sha256}")
        part.replace(f.path)
    except BaseException:
        part.unlink(missing_ok=True)
        raise
    return got


def _fetch_into(url: str, out: BinaryIO) -> str:
    """Writes the body at url to out, up to MAX_DOWNLOAD_BYTES, and returns its SHA-256."""
    digest = hashlib.sha256()
    size = 0
    # load_data_protocol checks the scheme of url; Hugging Face redirects to its CDN,
    # and urllib would follow a redirect from https to http.
    with urllib.request.urlopen(url, timeout=DOWNLOAD_TIMEOUT_S) as res:  # noqa: S310
        if urlparse(res.geturl()).scheme != urlparse(url).scheme:
            raise ValueError(f"{url}: redirected to {res.geturl()!r}, another scheme")
        while chunk := res.read(CHUNK_BYTES):
            size += len(chunk)
            if size > MAX_DOWNLOAD_BYTES:
                raise ValueError(f"{url}: larger than {MAX_DOWNLOAD_BYTES} bytes")
            digest.update(chunk)
            out.write(chunk)
    return digest.hexdigest()


def file_sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as f:
        while chunk := f.read(CHUNK_BYTES):
            digest.update(chunk)
    return digest.hexdigest()


def load_longmemeval(path: Path) -> list[Question]:
    """Reads a LongMemEval file; a missing field raises ValueError naming it."""
    records = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(records, list):
        raise ValueError(f"{path}: expected a list of questions")
    return [_question(path, i, r) for i, r in enumerate(records)]


def _question(path: Path, index: int, rec: dict[str, Any]) -> Question:
    label = rec.get("question_id", index) if isinstance(rec, dict) else index
    where = f"{path}: question {label}"
    _require(where, rec, QUESTION_FIELDS)
    ids, dates, turns = rec["haystack_session_ids"], rec["haystack_dates"], rec["haystack_sessions"]
    if not len(ids) == len(dates) == len(turns):
        raise ValueError(f"{where}: haystack ids, dates and sessions differ in length")
    sessions = [
        _session(where, str(sid), date, session)
        for sid, date, session in zip(ids, dates, turns, strict=True)
    ]
    qid = str(rec["question_id"])
    return Question(
        id=qid,
        type=rec["question_type"],
        question=rec["question"],
        # Some answers are numbers in the file.
        answer=str(rec["answer"]),
        question_date=rec["question_date"],
        sessions=sessions,
        evidence_session_ids=[str(s) for s in rec["answer_session_ids"]],
        is_abstention=qid.endswith(ABSTENTION_SUFFIX),
    )


def _session(where: str, sid: str, date: str, turns: list[Any]) -> Session:
    for turn in turns:
        _require(where, turn, TURN_FIELDS)
    return Session(
        id=sid,
        date=date,
        messages=[{"role": t["role"], "content": t["content"]} for t in turns],
        evidence_turns=[i for i, t in enumerate(turns) if t.get("has_answer")],
    )


def _require(where: str, rec: Any, fields: tuple[str, ...]) -> None:
    if not isinstance(rec, dict):
        raise ValueError(f"{where}: expected an object, got {type(rec).__name__}")
    missing = [f for f in fields if f not in rec]
    if missing:
        raise ValueError(f"{where}: missing field {', '.join(missing)}")


def split(questions: list[Question], name: str, seed: int) -> list[Question]:
    """Returns the questions of a split, in file order.

    dev60 and write100 are stratified by type, with abstention questions as
    strata of their own, and chosen by the SHA-256 of seed and question id;
    heldout440 is every question not in dev60.
    """
    if name == "all":
        return list(questions)
    if name == "heldout440":
        dev = {q.id for q in split(questions, "dev60", seed)}
        return [q for q in questions if q.id not in dev]
    if name not in SAMPLE_SIZES:
        raise ValueError(f"unknown split {name!r}; known splits are {SPLITS}")
    chosen = _stratified_sample(questions, SAMPLE_SIZES[name], seed)
    return [q for q in questions if q.id in chosen]


def _stratum(q: Question) -> str:
    return q.type + ABSTENTION_SUFFIX if q.is_abstention else q.type


def _stratified_sample(questions: list[Question], size: int, seed: int) -> set[str]:
    if size > len(questions):
        raise ValueError(f"cannot sample {size} of {len(questions)} questions")
    strata: dict[str, list[str]] = defaultdict(list)
    for q in questions:
        strata[_stratum(q)].append(q.id)
    quotas = _quotas({k: len(ids) for k, ids in strata.items()}, size)
    chosen: set[str] = set()
    for key, ids in strata.items():
        ranked = sorted(ids, key=lambda qid: _rank(seed, qid))
        chosen.update(ranked[: quotas[key]])
    return chosen


def _quotas(counts: dict[str, int], size: int) -> dict[str, int]:
    """Largest remainder apportionment of size over counts; ties go by stratum name."""
    total = sum(counts.values())
    quotas = {k: size * n // total for k, n in counts.items()}
    by_remainder = sorted(counts, key=lambda k: (-(size * counts[k] % total), k))
    for k in by_remainder[: size - sum(quotas.values())]:
        quotas[k] += 1
    return quotas


def _rank(seed: int, question_id: str) -> str:
    # A hash, not random.Random, so the split does not depend on the Python version.
    return hashlib.sha256(f"{seed}:{question_id}".encode()).hexdigest()


def session_repetition(questions: list[Question]) -> Repetition:
    """Counts session occurrences whose exact messages appear in another question too."""
    owners: dict[str, set[str]] = defaultdict(set)
    occurrences = []
    for q in questions:
        for s in q.sessions:
            key = hashlib.sha256(json.dumps(s.messages, ensure_ascii=False).encode()).hexdigest()
            owners[key].add(q.id)
            occurrences.append(key)
    repeated = sum(1 for key in occurrences if len(owners[key]) > 1)
    return Repetition(sessions=len(occurrences), repeated=repeated, distinct=len(owners))
