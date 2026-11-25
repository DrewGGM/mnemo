import json
import threading
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import pytest

from mnemo_eval import models

KEY = "test-key-never-shown"


class ScriptedHandler(BaseHTTPRequestHandler):
    """Answers each POST with the next (status, body) of the server's script.

    A status None closes the connection without answering; a bytes body is sent as is.
    """

    def do_POST(self):
        length = int(self.headers["Content-Length"])
        self.server.requests.append(
            (self.path, self.headers["Authorization"], json.loads(self.rfile.read(length)))
        )
        status, body = self.server.script.pop(0)
        if status is None:
            self.close_connection = True
            return
        data = body if isinstance(body, bytes) else json.dumps(body).encode()
        self.send_response(status)
        if status == HTTPStatus.FOUND:
            self.send_header("Location", "/leak")
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, format, *args):
        pass


@pytest.fixture
def endpoint():
    server = ThreadingHTTPServer(("127.0.0.1", 0), ScriptedHandler)
    server.requests, server.script = [], []
    thread = threading.Thread(target=server.serve_forever)
    thread.start()
    yield server
    server.shutdown()
    server.server_close()
    thread.join()


def chat(server, waits):
    url = f"http://127.0.0.1:{server.server_port}/v1/"
    return models.OpenAIChat(url, KEY, "reader-1", sleep=waits.append)


def ok(text="Toby", prompt_tokens=120, completion_tokens=3, total_tokens=None):
    usage = {"prompt_tokens": prompt_tokens, "completion_tokens": completion_tokens}
    if total_tokens is not None:
        usage["total_tokens"] = total_tokens
    return 200, {"choices": [{"message": {"content": text}}], "usage": usage}


def test_answer_and_provider_tokens_are_decoded(endpoint):
    endpoint.script = [ok()]
    got = chat(endpoint, []).complete("prompt", max_tokens=10)
    assert got == models.Completion("Toby", 120, 3)
    path, auth, body = endpoint.requests[0]
    assert (path, auth) == ("/v1/chat/completions", f"Bearer {KEY}")
    assert body == {
        "model": "reader-1",
        "messages": [{"role": "user", "content": "prompt"}],
        "temperature": 0,
        "max_tokens": 10,
    }


def test_output_tokens_include_thinking_left_out_of_completion_tokens(endpoint):
    endpoint.script = [ok(prompt_tokens=100, completion_tokens=5, total_tokens=145)]
    assert chat(endpoint, []).complete("p") == models.Completion("Toby", 100, 45)


@pytest.mark.parametrize("status", [429, 503])
def test_rate_limit_and_overload_are_retried_with_growing_waits(endpoint, status):
    endpoint.script = [(status, {}), (status, {}), ok()]
    waits = []
    assert chat(endpoint, waits).complete("p").text == "Toby"
    assert waits == list(models.BACKOFF_S[:2])


@pytest.mark.parametrize("failure", [(None, None), (200, b"<html>busy</html>")])
def test_dropped_connection_and_body_without_json_are_retried(endpoint, failure):
    endpoint.script = [failure, ok()]
    waits = []
    assert chat(endpoint, waits).complete("p").text == "Toby"
    assert waits == [models.BACKOFF_S[0]]


def test_dropped_connections_stop_after_the_last_wait(endpoint):
    endpoint.script = [(None, None)] * (len(models.BACKOFF_S) + 1)
    with pytest.raises(models.ModelError, match="RemoteDisconnected"):
        chat(endpoint, []).complete("p")


def test_retries_stop_after_the_last_wait(endpoint):
    endpoint.script = [(429, {})] * (len(models.BACKOFF_S) + 1)
    waits = []
    with pytest.raises(models.ModelError, match="HTTP 429"):
        chat(endpoint, waits).complete("p")
    assert waits == list(models.BACKOFF_S)


def test_empty_balance_is_not_retried(endpoint):
    body = {"error": {"message": "No credits", "type": "insufficient_quota"}}
    endpoint.script = [(429, body)]
    waits = []
    with pytest.raises(models.ModelError, match="HTTP 429 insufficient_quota"):
        chat(endpoint, waits).complete("p")
    assert waits == []


def test_bad_request_is_not_retried(endpoint):
    endpoint.script = [(400, {"error": {"code": "invalid_value", "message": "echo: prompt"}})]
    waits = []
    with pytest.raises(models.ModelError) as err:
        chat(endpoint, waits).complete("prompt")
    assert waits == []
    assert str(err.value).endswith("HTTP 400 invalid_value")


def test_gemini_error_list_gives_its_status(endpoint):
    endpoint.script = [(404, [{"error": {"code": 404, "status": "NOT_FOUND", "message": "m"}}])]
    with pytest.raises(models.ModelError, match="HTTP 404 NOT_FOUND"):
        chat(endpoint, []).complete("p")


def test_error_never_carries_the_key_or_the_body_message(endpoint):
    endpoint.script = [(401, {"error": {"message": f"bad key {KEY}", "code": "invalid_api_key"}})]
    with pytest.raises(models.ModelError) as err:
        chat(endpoint, []).complete("p")
    assert KEY not in str(err.value)
    assert "bad key" not in str(err.value)


def test_redirect_is_not_followed(endpoint):
    endpoint.script = [(302, {})]
    with pytest.raises(models.ModelError, match="HTTP 302"):
        chat(endpoint, []).complete("p")
    assert [path for path, _, _ in endpoint.requests] == ["/v1/chat/completions"]


def test_non_http_base_url_is_rejected():
    with pytest.raises(ValueError, match="http or https"):
        models.OpenAIChat("file:///etc", KEY, "m")


def test_fake_reader_answers_first_memory_line():
    prompt = "x\n<memory>\nline one\nline two\n</memory>\nQuestion: q"
    assert models.FakeReader().complete(prompt).text == "line one"
    assert models.FakeReader().complete("<memory>\n\n</memory>").text == models.NO_ANSWER
