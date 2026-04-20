import json
import threading
import urllib.request
import urllib.error

import pytest

from app import create_server, events_store


@pytest.fixture(autouse=True)
def _clear_store():
    events_store.clear()
    yield
    events_store.clear()


@pytest.fixture()
def server():
    srv = create_server(port=18001)
    t = threading.Thread(target=srv.serve_forever, daemon=True)
    t.start()
    yield srv
    srv.shutdown()


def _get(srv, path):
    url = f"http://localhost:{srv.server_address[1]}{path}"
    req = urllib.request.Request(url)
    with urllib.request.urlopen(req) as resp:
        return resp.status, json.loads(resp.read())


def _post(srv, path, data):
    url = f"http://localhost:{srv.server_address[1]}{path}"
    body = json.dumps(data).encode()
    req = urllib.request.Request(url, data=body, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req) as resp:
            return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read())


def test_health(server):
    status, body = _get(server, "/health")
    assert status == 200
    assert body["status"] == "healthy"
    assert body["service"] == "event-collector"


def test_post_event(server):
    status, body = _post(server, "/events", {"type": "click", "data": {"page": "/home"}})
    assert status == 201
    assert body["id"] == 1
    assert body["status"] == "collected"


def test_get_events(server):
    _post(server, "/events", {"type": "click", "data": {"page": "/home"}})
    _post(server, "/events", {"type": "view", "data": {"page": "/about"}})
    status, body = _get(server, "/events")
    assert status == 200
    assert body["count"] == 2
    assert len(body["events"]) == 2


def test_post_invalid_json(server):
    url = f"http://localhost:{server.server_address[1]}/events"
    req = urllib.request.Request(url, data=b"not json", headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req) as resp:
            status = resp.status
            body = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        status = e.code
        body = json.loads(e.read())
    assert status == 400
    assert "invalid JSON" in body["error"]


def test_post_missing_fields(server):
    status, body = _post(server, "/events", {"foo": "bar"})
    assert status == 400
    assert "type" in body["error"]


def test_not_found(server):
    url = f"http://localhost:{server.server_address[1]}/nonexistent"
    req = urllib.request.Request(url)
    try:
        with urllib.request.urlopen(req) as resp:
            status = resp.status
    except urllib.error.HTTPError as e:
        status = e.code
    assert status == 404
