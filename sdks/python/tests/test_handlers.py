"""Tests for handlers and execution context."""

import responses

from argyll import Client, StepContext, StepResult
from argyll.handlers import AsyncContext


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
