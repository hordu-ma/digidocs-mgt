import urllib.error
import urllib.request

import pytest

from app.clients.http_util import HttpError, fetch


class _Resp:
    def __init__(self, body: bytes = b"ok", status: int = 200) -> None:
        self._body = body
        self.status = status
        self.headers = {"Content-Type": "application/json"}

    def read(self) -> bytes:
        return self._body

    def __enter__(self):
        return self

    def __exit__(self, *_a) -> None:
        return None


def _req() -> urllib.request.Request:
    return urllib.request.Request("http://x/y", method="GET")


def test_fetch_success_no_retry(monkeypatch) -> None:
    calls = {"n": 0}

    def fake(request, timeout):
        calls["n"] += 1
        return _Resp(b'{"ok":true}')

    monkeypatch.setattr("urllib.request.urlopen", fake)
    status, headers, body = fetch(_req(), timeout=5, label="t", sleep=lambda _s: None)
    assert status == 200 and body == b'{"ok":true}'
    assert calls["n"] == 1


def test_fetch_retries_transient_then_succeeds(monkeypatch) -> None:
    calls = {"n": 0}

    def fake(request, timeout):
        calls["n"] += 1
        if calls["n"] < 3:
            raise urllib.error.URLError("connection refused")
        return _Resp(b"recovered")

    monkeypatch.setattr("urllib.request.urlopen", fake)
    _, _, body = fetch(_req(), timeout=5, label="t", attempts=3, sleep=lambda _s: None)
    assert body == b"recovered"
    assert calls["n"] == 3


def test_fetch_raises_after_exhausting_transient(monkeypatch) -> None:
    calls = {"n": 0}

    def fake(request, timeout):
        calls["n"] += 1
        raise urllib.error.URLError("down")

    monkeypatch.setattr("urllib.request.urlopen", fake)
    with pytest.raises(urllib.error.URLError):
        fetch(_req(), timeout=5, label="t", attempts=2, sleep=lambda _s: None)
    assert calls["n"] == 2


def test_fetch_does_not_retry_4xx(monkeypatch) -> None:
    calls = {"n": 0}

    def fake(request, timeout):
        calls["n"] += 1
        raise urllib.error.HTTPError("http://x/y", 404, "Not Found", {}, None)  # type: ignore[arg-type]

    monkeypatch.setattr("urllib.request.urlopen", fake)
    with pytest.raises(HttpError) as ei:
        fetch(_req(), timeout=5, label="t", attempts=3, sleep=lambda _s: None)
    assert ei.value.status == 404
    assert calls["n"] == 1


def test_fetch_retries_5xx_then_raises_http_error(monkeypatch) -> None:
    calls = {"n": 0}

    def fake(request, timeout):
        calls["n"] += 1
        raise urllib.error.HTTPError("http://x/y", 503, "Unavailable", {}, None)  # type: ignore[arg-type]

    monkeypatch.setattr("urllib.request.urlopen", fake)
    with pytest.raises(HttpError) as ei:
        fetch(_req(), timeout=5, label="t", attempts=3, sleep=lambda _s: None)
    assert ei.value.status == 503
    assert calls["n"] == 3
