"""Tests for Client and FlowClient."""

import responses

from argyll import Client
from argyll.errors import ClientError, FlowError
from argyll.types import StepType


def test_client_initialization():
    client = Client()
    assert client.base_url == "http://localhost:8080"
    assert client.timeout == 30


def test_client_custom_url():
    client = Client(base_url="http://example.com:9000/api")
    assert client.base_url == "http://example.com:9000/api"


def test_client_strips_engine_suffix():
    client = Client(base_url="http://localhost:8080/engine")
    assert client.base_url == "http://localhost:8080"


@responses.activate
def test_list_steps_empty():
    responses.add(
        responses.GET,
        "http://localhost:8080/engine/step",
        json={"steps": []},
        status=200,
    )

    client = Client()
    steps = client.list_steps()
    assert steps == []


@responses.activate
def test_list_steps_with_data():
    responses.add(
        responses.GET,
        "http://localhost:8080/engine/step",
        json={
            "steps": [
                {
                    "id": "step-1",
                    "name": "Step 1",
                    "type": "sync",
                    "attributes": {
                        "input": {
                            "role": "required",
                            "type": "string",
                        }
                    },
                }
            ],
            "count": 1,
        },
        status=200,
    )

    client = Client()
    steps = client.list_steps()
    assert len(steps) == 1
    assert steps[0].id == "step-1"
    assert steps[0].name == "Step 1"
    assert steps[0].type == StepType.SYNC


@responses.activate
def test_register_step():
    responses.add(
        responses.POST,
        "http://localhost:8080/engine/step",
        json={},
        status=200,
    )

    from argyll.types import HTTPConfig, Step

    client = Client()
    step = Step(
        id="test-step",
        name="Test",
        type=StepType.SYNC,
        http=HTTPConfig(endpoint="http://localhost:8081/test"),
    )
    client.register_step(step)

    assert len(responses.calls) == 1
    assert responses.calls[0].request.url == (
        "http://localhost:8080/engine/step"
    )


@responses.activate
def test_register_step_error():
    responses.add(
        responses.POST,
        "http://localhost:8080/engine/step",
        json={"error": "Invalid step"},
        status=400,
    )

    from argyll.types import HTTPConfig, Step

    client = Client()
    step = Step(
        id="test-step",
        name="Test",
        type=StepType.SYNC,
        http=HTTPConfig(endpoint="http://localhost:8081/test"),
    )

    try:
        client.register_step(step)
        assert False, "Should have raised ClientError"
    except ClientError as e:
        assert e.status_code == 400


@responses.activate
def test_update_step():
    responses.add(
        responses.PUT,
        "http://localhost:8080/engine/step/test-step",
        json={},
        status=200,
    )

    from argyll.types import HTTPConfig, Step

    client = Client()
    step = Step(
        id="test-step",
        name="Test",
        type=StepType.SYNC,
        http=HTTPConfig(endpoint="http://localhost:8081/test"),
    )
    client.update_step(step)

    assert len(responses.calls) == 1


def test_new_step():
    client = Client()
    builder = client.new_step("Test Step")
    assert builder._name == "Test Step"
    assert builder._id == "test-step"


def test_new_flow():
    client = Client()
    builder = client.new_flow("flow-123")
    assert builder._flow_id == "flow-123"


def test_flow_client():
    client = Client()
    flow_client = client.flow("flow-123")
    assert flow_client.flow_id == "flow-123"


@responses.activate
def test_flow_client_get_state():
    responses.add(
        responses.GET,
        "http://localhost:8080/engine/flow/flow-123",
        json={"status": "active", "attributes": {}},
        status=200,
    )

    client = Client()
    flow_client = client.flow("flow-123")
    state = flow_client.get_state()
    assert state["status"] == "active"


@responses.activate
def test_flow_client_get_state_error():
    responses.add(
        responses.GET,
        "http://localhost:8080/engine/flow/flow-123",
        json={"error": "Not found"},
        status=404,
    )

    client = Client()
    flow_client = client.flow("flow-123")

    try:
        flow_client.get_state()
        assert False, "Should have raised FlowError"
    except FlowError:
        pass


@responses.activate
def test_list_steps_error():
    responses.add(
        responses.GET,
        "http://localhost:8080/engine/step",
        json={"error": "Server error"},
        status=500,
    )

    client = Client()
    try:
        client.list_steps()
        assert False, "Should have raised ClientError"
    except ClientError as e:
        assert e.status_code == 500


@responses.activate
def test_parse_step_with_all_fields():
    responses.add(
        responses.GET,
        "http://localhost:8080/engine/step",
        json={
            "steps": [
                {
                    "id": "complex-step",
                    "name": "Complex Step",
                    "type": "async",
                    "attributes": {
                        "input": {"role": "required", "type": "string"},
                        "output": {"role": "output", "type": "number"},
                    },
                    "labels": {"env": "prod"},
                    "http": {
                        "endpoint": "http://localhost:8081/complex",
                        "health_check": "http://localhost:8081/health",
                        "timeout": 5000,
                    },
                    "script": {"language": "ale", "script": "(+ 1 2)"},
                    "predicate": {"language": "lua", "script": "return true"},
                    "work_config": {
                        "max_retries": 3,
                        "backoff_type": "exponential",
                        "backoff": 1000,
                        "max_backoff": 10000,
                        "parallelism": 2,
                    },
                    "flow": {
                        "goals": ["step-1", "step-2"],
                        "input_map": {"a": "b"},
                        "output_map": {"c": "d"},
                    },
                    "memoizable": True,
                }
            ],
            "count": 1,
        },
        status=200,
    )

    client = Client()
    steps = client.list_steps()
    assert len(steps) == 1
    step = steps[0]
    assert step.id == "complex-step"
    assert step.http is not None
    assert step.http.timeout == 5000
    assert step.script is not None
    assert step.predicate is not None
    assert step.work_config is not None
    assert step.flow is not None
    assert step.memoizable is True
