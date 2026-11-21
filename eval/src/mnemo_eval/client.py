"""HTTP client for the mnemo service, on the standard library."""

import json
import urllib.error
import urllib.request
from typing import Any, override
from urllib.parse import urlsplit

# The server waits up to 60 s for extraction when wait=true.
DEFAULT_TIMEOUT_S = 70.0


class MnemoError(Exception):
    """A request to mnemo failed or returned an error status."""


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    """Turns a 3xx into an HTTPError: urllib would resend Authorization to any Location."""

    @override
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class MnemoClient:
    """Calls the /v1 routes of one mnemo service."""

    def __init__(self, base_url: str, api_key: str | None = None) -> None:
        # urllib openers also open file:// and ftp:// URLs.
        if urlsplit(base_url).scheme not in ("http", "https"):
            raise ValueError("base_url must be an http or https URL")
        self._base_url = base_url.rstrip("/")
        self._api_key = api_key
        self._opener = urllib.request.build_opener(_NoRedirect)

    def remember(
        self, subject: str, session_id: str, messages: list[dict[str, Any]], wait: bool = True
    ) -> dict[str, Any]:
        """Stores messages of one session; with wait, returns once extraction ends."""
        path = "/v1/episodes?wait=true" if wait else "/v1/episodes"
        return self._post(
            path, {"subject": subject, "session_id": session_id, "messages": messages}
        )

    def recall(self, subject: str, query: str, **params: Any) -> dict[str, Any]:
        """Returns the memory of subject for query; params go to the request body."""
        return self._post("/v1/recall", {**params, "subject": subject, "query": query})

    def _post(self, path: str, body: dict[str, Any]) -> dict[str, Any]:
        headers = {"Content-Type": "application/json"}
        if self._api_key is not None:
            headers["Authorization"] = f"Bearer {self._api_key}"
        req = urllib.request.Request(  # noqa: S310 - the scheme is checked in __init__
            self._base_url + path,
            data=json.dumps(body).encode(),
            headers=headers,
            method="POST",
        )
        try:
            with self._opener.open(req, timeout=DEFAULT_TIMEOUT_S) as resp:
                return json.load(resp)
        except urllib.error.HTTPError as err:
            with err:
                detail = _error_detail(err.read())
            raise MnemoError(f"POST {path}: HTTP {err.code}: {detail}") from None
        except (urllib.error.URLError, TimeoutError) as err:
            raise MnemoError(f"POST {path}: {err}") from None


def _error_detail(body: bytes) -> str:
    """Returns the code and message of mnemo's error envelope."""
    try:
        error = json.loads(body)["error"]
        return f"{error['code']}: {error['message']}"
    except (ValueError, KeyError, TypeError):
        return "response without an error envelope"
