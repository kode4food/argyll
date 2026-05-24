"""Tests for handlers and execution context."""

import pytest
import responses

from argyll import Client, StepContext, handlers
from argyll.builder import StepBuilder
from argyll.errors import ClientError, HTTPError, WebhookError
from argyll.handlers import AsyncContext, _execute_with_recovery


def test_step_context_creation():
    client = Client()
    flow_client = client.flow("flow-123")
    ctx = StepContext(
        client=flow_client, step_id="step-1", metadata={"key": "value"}
    )
    assert ctx.client.flow_id == "flow-123"
    assert ctx.step_id == "step-1"
    assert ctx.metadata["key"] == "value"


def test_async_context_creation():
    client = Client()
    flow_client = client.flow("flow-123")
    step_ctx = StepContext(
        client=flow_client,
        step_id="step-1",
        metadata={"webhook_url": "http://localhost:8080/webhook"},
    )
    async_ctx = AsyncContext(
        context=step_ctx, webhook_url="http://localhost:8080/webhook"
    )
    assert async_ctx.flow_id == "flow-123"
    assert async_ctx.step_id == "step-1"
    assert async_ctx.webhook_url == "http://localhost:8080/webhook"


def test_async_context_properties():
    client = Client()
    flow_client = client.flow("flow-123")
    step_ctx = StepContext(
        client=flow_client,
        step_id="step-1",
        metadata={"webhook_url": "http://localhost:8080/webhook"},
    )
    async_ctx = AsyncContext(
        context=step_ctx, webhook_url="http://localhost:8080/webhook"
    )
    assert async_ctx.client is flow_client
    assert async_ctx.metadata["webhook_url"] == "http://localhost:8080/webhook"


@responses.activate
def test_async_context_success():
    responses.add(
        responses.POST,
        "http://localhost:8080/webhook",
        json={},
        status=200,
    )

    client = Client()
    flow_client = client.flow("flow-123")
    step_ctx = StepContext(
        client=flow_client,
        step_id="step-1",
        metadata={"webhook_url": "http://localhost:8080/webhook"},
    )
    async_ctx = AsyncContext(
        context=step_ctx, webhook_url="http://localhost:8080/webhook"
    )

    async_ctx.success({"result": "done"})

    assert len(responses.calls) == 1
    import json

    req_body = json.loads(responses.calls[0].request.body)
    assert req_body["result"] == "done"


@responses.activate
def test_async_context_fail():
    responses.add(
        responses.POST,
        "http://localhost:8080/webhook",
        json={},
        status=200,
    )

    client = Client()
    flow_client = client.flow("flow-123")
    step_ctx = StepContext(
        client=flow_client,
        step_id="step-1",
        metadata={"webhook_url": "http://localhost:8080/webhook"},
    )
    async_ctx = AsyncContext(
        context=step_ctx, webhook_url="http://localhost:8080/webhook"
    )

    async_ctx.fail("Something went wrong")

    assert len(responses.calls) == 1
    import json

    req_body = json.loads(responses.calls[0].request.body)
    assert responses.calls[0].request.headers["Content-Type"] == (
        "application/problem+json"
    )
    assert req_body["detail"] == "Something went wrong"
    assert req_body["status"] == 422


@responses.activate
def test_async_context_complete():
    responses.add(
        responses.POST,
        "http://localhost:8080/webhook",
        json={},
        status=200,
    )

    client = Client()
    flow_client = client.flow("flow-123")
    step_ctx = StepContext(
        client=flow_client,
        step_id="step-1",
        metadata={"webhook_url": "http://localhost:8080/webhook"},
    )
    async_ctx = AsyncContext(
        context=step_ctx, webhook_url="http://localhost:8080/webhook"
    )

    async_ctx.complete({"data": "value"})

    assert len(responses.calls) == 1
    import json

    req_body = json.loads(responses.calls[0].request.body)
    assert req_body["data"] == "value"


@responses.activate
def test_async_context_webhook_error():
    responses.add(
        responses.POST,
        "http://localhost:8080/webhook",
        json={"error": "fail"},
        status=500,
    )

    client = Client()
    flow_client = client.flow("flow-123")
    step_ctx = StepContext(
        client=flow_client,
        step_id="step-1",
        metadata={"webhook_url": "http://localhost:8080/webhook"},
    )
    async_ctx = AsyncContext(
        context=step_ctx, webhook_url="http://localhost:8080/webhook"
    )

    with pytest.raises(WebhookError):
        async_ctx.success({"result": "done"})


def test_execute_with_recovery_success():
    client = Client()
    ctx = StepContext(
        client=client.flow("flow-123"),
        step_id="step-1",
        metadata={},
    )

    def handler(step_ctx, args):
        return {"value": args["value"]}

    result = _execute_with_recovery(ctx, handler, {"value": 1})
    assert result["value"] == 1


def test_execute_with_recovery_http_error():
    client = Client()
    ctx = StepContext(
        client=client.flow("flow-123"),
        step_id="step-1",
        metadata={},
    )

    def handler(step_ctx, args):
        raise HTTPError(422, "bad input")

    with pytest.raises(HTTPError):
        _execute_with_recovery(ctx, handler, {})


def test_execute_with_recovery_exception_returns_failure():
    client = Client()
    ctx = StepContext(
        client=client.flow("flow-123"),
        step_id="step-1",
        metadata={},
    )

    def handler(step_ctx, args):
        raise ValueError("boom")

    with pytest.raises(HTTPError) as exc:
        _execute_with_recovery(ctx, handler, {})
    assert "panicked" in str(exc.value)


class _DummyFlowClient:
    def __init__(self, flow_id: str) -> None:
        self.flow_id = flow_id


class _DummyClient:
    def __init__(self) -> None:
        self.registered = []
        self.updated = []
        self.flow_ids = []

    def register_step(self, step):
        self.registered.append(step)

    def update_step(self, step):
        self.updated.append(step)

    def flow(self, flow_id: str):
        self.flow_ids.append(flow_id)
        return _DummyFlowClient(flow_id)


def test_create_step_server_registers_and_handles_request(monkeypatch):
    captured = {}

    def fake_run(self, host, port):
        captured["app"] = self
        captured["host"] = host
        captured["port"] = port

    monkeypatch.setenv("STEP_PORT", "9010")
    monkeypatch.setenv("STEP_HOSTNAME", "example.com")
    monkeypatch.setattr(handlers.Flask, "run", fake_run, raising=True)

    client = _DummyClient()
    builder = StepBuilder(client=client, name="Test Step")

    def handler(step_ctx, args):
        return {"value": args["value"]}

    handlers.create_step_server(client, builder, handler)

    assert len(client.registered) == 1
    app = captured["app"]
    test_client = app.test_client()

    resp = test_client.get("/health")
    assert resp.status_code == 200
    assert resp.get_json()["service"] == "test-step"

    resp = test_client.post(
        "/test-step",
        json={"value": 3},
        headers={"Argyll-Flow-ID": "flow-1"},
    )
    assert resp.status_code == 200
    data = resp.get_json()
    assert data["value"] == 3
    assert client.flow_ids == ["flow-1"]

    bad_resp = test_client.post(
        "/test-step", data="null", content_type="application/json"
    )
    assert bad_resp.status_code == 400


def test_create_step_server_http_error(monkeypatch):
    captured = {}

    def fake_run(self, host, port):
        captured["app"] = self

    monkeypatch.setattr(handlers.Flask, "run", fake_run, raising=True)

    client = _DummyClient()
    builder = StepBuilder(client=client, name="Test Step")

    def handler(step_ctx, args):
        raise HTTPError(409, "conflict")

    handlers.create_step_server(client, builder, handler)

    app = captured["app"]
    test_client = app.test_client()
    resp = test_client.post(
        "/test-step",
        json={},
        headers={"Argyll-Flow-ID": "flow-1"},
    )
    assert resp.status_code == 409
    assert resp.get_json()["detail"] == "conflict"


def test_create_step_server_update(monkeypatch):
    captured = {}

    def fake_run(self, host, port):
        captured["app"] = self

    monkeypatch.setattr(handlers.Flask, "run", fake_run, raising=True)

    client = _DummyClient()
    builder = StepBuilder(client=client, name="Test Step").update()

    def handler(step_ctx, args):
        return {}

    handlers.create_step_server(client, builder, handler)

    assert len(client.updated) == 1
    assert len(client.registered) == 0


def test_step_server_register_conflict_updates(monkeypatch):
    captured = {}

    def fake_run(self, host, port):
        captured["app"] = self

    monkeypatch.setattr(handlers.Flask, "run", fake_run, raising=True)

    class _ConflictClient(_DummyClient):
        def register_step(self, step):
            self.registered.append(step)
            raise ClientError("conflict", status_code=409)

    client = _ConflictClient()
    builder = StepBuilder(client=client, name="Test Step")

    def handler(step_ctx, args):
        return {}

    handlers.create_step_server(client, builder, handler)

    assert len(client.registered) == 1
    assert len(client.updated) == 1


def test_step_server_unhandled_exception(monkeypatch):
    captured = {}

    def fake_run(self, host, port):
        captured["app"] = self

    monkeypatch.setattr(handlers.Flask, "run", fake_run, raising=True)

    class _BoomClient(_DummyClient):
        def flow(self, flow_id):
            raise RuntimeError("unexpected boom from flow()")

    client = _BoomClient()
    builder = StepBuilder(client=client, name="Test Step")

    handlers.create_step_server(client, builder, lambda ctx, args: {})

    app = captured["app"]
    test_client = app.test_client()
    resp = test_client.post(
        "/test-step",
        json={},
        headers={"Argyll-Flow-ID": "flow-1"},
    )
    assert resp.status_code == 500
    assert resp.get_json()["error"] == "Internal server error"


def test_step_server_compensate_handler(monkeypatch):
    captured = {}

    def fake_run(self, host, port):
        captured["app"] = self

    monkeypatch.setenv("STEP_PORT", "9020")
    monkeypatch.setenv("STEP_HOSTNAME", "localhost")
    monkeypatch.setattr(handlers.Flask, "run", fake_run, raising=True)

    client = _DummyClient()
    builder = StepBuilder(client=client, name="Test Step")

    comp_calls = []

    def handler(step_ctx, args):
        return {"done": True}

    def compensate_handler(step_ctx, inputs, outputs):
        comp_calls.append((inputs, outputs))

    handlers.create_step_server(client, builder, handler, compensate_handler)

    step = client.registered[0]
    assert step.http is not None
    assert step.http.compensate == "http://localhost:9020/test-step/compensate"

    app = captured["app"]
    test_client = app.test_client()

    resp = test_client.post(
        "/test-step/compensate",
        json={"input": {"amount": 100}, "output": {"ref": "abc"}},
        headers={"Argyll-Flow-ID": "flow-1"},
    )
    assert resp.status_code == 204
    assert len(comp_calls) == 1
    assert comp_calls[0] == ({"amount": 100}, {"ref": "abc"})


def test_step_server_compensate_bad_json(monkeypatch):
    captured = {}

    def fake_run(self, host, port):
        captured["app"] = self

    monkeypatch.setattr(handlers.Flask, "run", fake_run, raising=True)

    client = _DummyClient()
    builder = StepBuilder(client=client, name="Test Step")

    handlers.create_step_server(
        client, builder, lambda ctx, args: {}, lambda ctx, i, o: None
    )

    app = captured["app"]
    test_client = app.test_client()
    resp = test_client.post(
        "/test-step/compensate",
        data="null",
        content_type="application/json",
    )
    assert resp.status_code == 400


def test_step_server_compensate_http_error(monkeypatch):
    captured = {}

    def fake_run(self, host, port):
        captured["app"] = self

    monkeypatch.setattr(handlers.Flask, "run", fake_run, raising=True)

    client = _DummyClient()
    builder = StepBuilder(client=client, name="Test Step")

    def compensate_handler(step_ctx, inputs, outputs):
        raise HTTPError(422, "cannot undo")

    handlers.create_step_server(
        client, builder, lambda ctx, args: {}, compensate_handler
    )

    app = captured["app"]
    test_client = app.test_client()
    resp = test_client.post(
        "/test-step/compensate",
        json={"input": {}, "output": {}},
        headers={"Argyll-Flow-ID": "flow-1"},
    )
    assert resp.status_code == 422
    assert resp.get_json()["detail"] == "cannot undo"


def test_step_server_compensate_exception(monkeypatch):
    captured = {}

    def fake_run(self, host, port):
        captured["app"] = self

    monkeypatch.setattr(handlers.Flask, "run", fake_run, raising=True)

    client = _DummyClient()
    builder = StepBuilder(client=client, name="Test Step")

    def compensate_handler(step_ctx, inputs, outputs):
        raise RuntimeError("comp boom")

    handlers.create_step_server(
        client, builder, lambda ctx, args: {}, compensate_handler
    )

    app = captured["app"]
    test_client = app.test_client()
    resp = test_client.post(
        "/test-step/compensate",
        json={"input": {}, "output": {}},
        headers={"Argyll-Flow-ID": "flow-1"},
    )
    assert resp.status_code == 500
    data = resp.get_json()
    assert data["detail"] == "Internal server error"


def test_step_server_register_retry(monkeypatch):
    captured = {}

    def fake_run(self, host, port):
        captured["app"] = self

    monkeypatch.setattr(handlers.Flask, "run", fake_run, raising=True)
    monkeypatch.setattr(handlers.time, "sleep", lambda _: None)

    attempt_count = 0

    class _RetryClient(_DummyClient):
        def register_step(self, step):
            nonlocal attempt_count
            attempt_count += 1
            if attempt_count < 3:
                raise ClientError("temporary", status_code=503)
            self.registered.append(step)

    client = _RetryClient()
    builder = StepBuilder(client=client, name="Test Step")

    handlers.create_step_server(client, builder, lambda ctx, args: {})

    assert attempt_count == 3
    assert len(client.registered) == 1


def test_step_server_register_exhausted(monkeypatch):
    monkeypatch.setattr(handlers.Flask, "run", lambda self, host, port: None)
    monkeypatch.setattr(handlers.time, "sleep", lambda _: None)

    class _AlwaysFailClient(_DummyClient):
        def register_step(self, step):
            raise ClientError("always fails", status_code=503)

    client = _AlwaysFailClient()
    builder = StepBuilder(client=client, name="Test Step")

    with pytest.raises(ClientError):
        handlers.create_step_server(client, builder, lambda ctx, args: {})
