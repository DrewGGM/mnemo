"""Chat models of the harness: an OpenAI-compatible client on the standard
library, and the deterministic fakes of the minimal evaluation."""

import http.client
import json
import os
import time
import urllib.error
import urllib.request
from collections.abc import Callable, Mapping
from dataclasses import dataclass
from http import HTTPStatus
from typing import Any, Protocol
from urllib.parse import urlsplit

from mnemo_eval.arms import estimate_tokens
from mnemo_eval.client import _NoRedirect

FAKE = "fake"
NO_ANSWER = "I don't know"
# The reader prompt wraps the memory in these lines; the fake reader reads them.
MEMORY_OPEN = "<memory>\n"
MEMORY_CLOSE = "\n</memory>"
# A full LongMemEval_S history is about 115,000 tokens in one prompt.
TIMEOUT_S = 180.0
RETRY_STATUS = frozenset({429, 500, 502, 503, 504})
# The free tier of the reader answers 429 for a minute once its requests per
# minute run out, and 503 while the model is in high demand.
BACKOFF_S = (2, 4, 8, 16, 32, 60, 60, 60)
# A 429 for an empty balance: waiting does not refill it.
QUOTA_ERRORS = frozenset({"insufficient_quota", "credit_balance_exhausted"})
# The status of an attempt that got no HTTP answer, or a 200 that is not JSON.
NO_ANSWER_STATUS = 0


@dataclass(frozen=True)
class Completion:
    """An answer and the tokens the provider reports for it."""

    text: str
    input_tokens: int
    output_tokens: int


class ChatModel(Protocol):
    def complete(self, prompt: str, max_tokens: int | None = None) -> Completion:
        """Answers prompt, sent as one user message, with temperature 0."""
        ...


class ModelError(Exception):
    """A chat request failed; the message never carries the key or the content."""


class OpenAIChat:
    """POST /chat/completions of an OpenAI-compatible endpoint."""

    def __init__(
        self,
        base_url: str,
        api_key: str,
        model: str,
        sleep: Callable[[float], None] = time.sleep,
    ) -> None:
        parts = urlsplit(base_url)
        # urllib openers also open file:// and ftp:// URLs.
        if parts.scheme not in ("http", "https") or not parts.netloc:
            raise ValueError(f"base URL of {model} must be an http or https URL")
        self._url = base_url.rstrip("/") + "/chat/completions"
        self._where = f"{model} at {parts.netloc}"
        self._api_key = api_key
        self.model = model
        self._sleep = sleep
        # A redirect would resend the key to the host of Location.
        self._opener = urllib.request.build_opener(_NoRedirect)

    def complete(self, prompt: str, max_tokens: int | None = None) -> Completion:
        body: dict[str, Any] = {
            "model": self.model,
            "messages": [{"role": "user", "content": prompt}],
            "temperature": 0,
        }
        if max_tokens is not None:
            body["max_tokens"] = max_tokens
        retries = 0
        while True:
            status, payload = self._post(body)
            if status == HTTPStatus.OK:
                return self._completion(payload)
            if status == NO_ANSWER_STATUS:
                failure, retry = f"{self._where}: {payload}", True
            else:
                code = _error_code(payload)
                failure = f"{self._where}: HTTP {status} {code}".rstrip()
                retry = status in RETRY_STATUS and code not in QUOTA_ERRORS
            if not retry or retries == len(BACKOFF_S):
                raise ModelError(failure)
            self._sleep(BACKOFF_S[retries])
            retries += 1

    def _post(self, body: dict[str, Any]) -> tuple[int, Any]:
        """The status and JSON body, or NO_ANSWER_STATUS and the name of the error."""
        req = urllib.request.Request(  # noqa: S310 - the scheme is checked in __init__
            self._url,
            data=json.dumps(body).encode(),
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {self._api_key}",
            },
            method="POST",
        )
        try:
            return self._send(req)
        # Refused or reset connections, timeouts and cut or non-JSON bodies:
        # urllib wraps only the errors of sending the request in URLError.
        except (OSError, http.client.HTTPException, ValueError) as err:
            return NO_ANSWER_STATUS, type(err).__name__

    def _send(self, req: urllib.request.Request) -> tuple[int, Any]:
        try:
            with self._opener.open(req, timeout=TIMEOUT_S) as resp:
                return resp.status, json.load(resp)
        except urllib.error.HTTPError as err:
            with err:
                return err.code, _json_or_none(err.read())

    def _completion(self, payload: Any) -> Completion:
        try:
            text = payload["choices"][0]["message"].get("content") or ""
            usage = payload["usage"]
            prompt_tokens = int(usage["prompt_tokens"])
            # Gemini leaves thinking tokens out of completion_tokens and bills
            # them as output; total_tokens counts them under both conventions.
            total = int(usage.get("total_tokens") or prompt_tokens + usage["completion_tokens"])
        except (KeyError, IndexError, TypeError, ValueError, AttributeError):
            raise ModelError(f"{self._where}: response without choices or usage") from None
        return Completion(
            text=text, input_tokens=prompt_tokens, output_tokens=total - prompt_tokens
        )


def _json_or_none(body: bytes) -> Any:
    try:
        return json.loads(body)
    except ValueError:
        return None


def _error_code(payload: Any) -> str:
    """The short code of an error body: OpenAI sends an object, Gemini a list of one.

    The message is left out: it may repeat what was sent.
    """
    if isinstance(payload, list) and payload:
        payload = payload[0]
    error = payload.get("error") if isinstance(payload, dict) else None
    if not isinstance(error, dict):
        return ""
    for key in ("code", "type", "status"):
        value = error.get(key)
        if isinstance(value, str) and value:
            return value
    return ""


class FakeReader:
    """Answers with the first line of the memory block, or NO_ANSWER without memory.

    Tokens are estimated as characters / 4, like the service does.
    """

    def complete(self, prompt: str, max_tokens: int | None = None) -> Completion:
        start = prompt.index(MEMORY_OPEN) + len(MEMORY_OPEN)
        memory = prompt[start : prompt.index(MEMORY_CLOSE, start)].strip()
        text = memory.splitlines()[0] if memory else NO_ANSWER
        return Completion(
            text=text, input_tokens=estimate_tokens(prompt), output_tokens=estimate_tokens(text)
        )


class FakeJudge:
    """Says yes when the reference answer of the judge prompt appears in the response."""

    REFERENCES = ("Correct Answer: ", "Rubric: ", "Explanation: ")
    RESPONSE = "\n\nModel Response: "

    def complete(self, prompt: str, max_tokens: int | None = None) -> Completion:
        head, _, tail = prompt.partition(self.RESPONSE)
        response = tail.rsplit("\n\n", 1)[0]
        reference = next(
            head.rsplit(label, 1)[1] for label in self.REFERENCES if label in head
        ).strip()
        text = "yes" if reference.casefold() in response.casefold() else "no"
        return Completion(text=text, input_tokens=estimate_tokens(prompt), output_tokens=1)


def open_model(
    model: str, env_prefix: str, fake: ChatModel, env: Mapping[str, str] = os.environ
) -> ChatModel:
    """Returns fake for the model "fake", else the endpoint that env_prefix names.

    Reads <env_prefix>_BASE_URL and _API_KEY; <env_prefix>_MODEL, when set, must
    be the protocol's model, so that a run never answers under another id.
    """
    if model == FAKE:
        return fake
    base_url = env.get(f"{env_prefix}_BASE_URL", "")
    if not base_url:
        raise ValueError(f"{env_prefix}_BASE_URL is not set; it serves {model}")
    env_model = env.get(f"{env_prefix}_MODEL", "")
    if env_model and env_model != model:
        raise ValueError(f"{env_prefix}_MODEL is {env_model!r} but the protocol fixes {model!r}")
    return OpenAIChat(base_url, env.get(f"{env_prefix}_API_KEY", ""), model)
