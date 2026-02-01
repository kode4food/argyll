"""Tests for handlers and execution context."""

import pytest
import responses

from argyll import Client, StepContext, StepResult, handlers
from argyll.builder import StepBuilder
from argyll.errors import HTTPError, WebhookError
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
    assert req_body["success"] is True
    assert req_body["outputs"]["result"] == "done"


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
    assert req_body["success"] is False
    assert req_body["error"] == "Something went wrong"


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

    result = StepResult(success=True, outputs={"data": "value"})
    async_ctx.complete(result)

    assert len(responses.calls) == 1
    import json

    req_body = json.loads(responses.calls[0].request.body)
    assert req_body["success"] is True
    assert req_body["outputs"]["data"] == "value"


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
        return StepResult(success=True, outputs={"value": args["value"]})

    result = _execute_with_recovery(ctx, handler, {"value": 1})
    assert result.success is True
    assert result.outputs["value"] == 1


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

    result = _execute_with_recovery(ctx, handler, {})
    assert result.success is False
    assert "panicked" in result.error


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
        return StepResult(success=True, outputs={"value": args["value"]})

    handlers.create_step_server(client, builder, handler)

    assert len(client.registered) == 1
    app = captured["app"]
    test_client = app.test_client()

    resp = test_client.get("/health")
    assert resp.status_code == 200
    assert resp.get_json()["service"] == "test-step"

    resp = test_client.post(
        "/test-step",
        json={"arguments": {"value": 3}, "metadata": {"flow_id": "flow-1"}},
    )
    assert resp.status_code == 200
    data = resp.get_json()
    assert data["success"] is True
    assert data["outputs"]["value"] == 3
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
        json={"arguments": {}, "metadata": {"flow_id": "flow-1"}},
    )
    assert resp.status_code == 409
    assert resp.get_json()["error"] == "conflict"


def test_create_step_server_update(monkeypatch):
    captured = {}

    def fake_run(self, host, port):
        captured["app"] = self

    monkeypatch.setattr(handlers.Flask, "run", fake_run, raising=True)

    client = _DummyClient()
    builder = StepBuilder(client=client, name="Test Step").update()

    def handler(step_ctx, args):
        return StepResult(success=True)

    handlers.create_step_server(client, builder, handler)

    assert len(client.updated) == 1
    assert len(client.registered) == 0
