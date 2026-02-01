"""Tests for StepBuilder and FlowBuilder."""

import responses

from argyll import Client
from argyll.errors import StepValidationError
from argyll.types import AttributeRole, AttributeType, ScriptLanguage, StepType


def test_step_builder_initialization():
    client = Client()
    builder = client.new_step("Test Step")
    assert builder._name == "Test Step"
    assert builder._id == "test-step"
    assert builder._type == StepType.SYNC


def test_step_builder_with_id():
    client = Client()
    b1 = client.new_step("Test")
    b2 = b1.with_id("custom-id")
    assert b1._id == "test"
    assert b2._id == "custom-id"


def test_step_builder_immutability():
    client = Client()
    b1 = client.new_step("Test")
    b2 = b1.with_id("custom")
    assert b1._id != b2._id
    assert b1._id == "test"
    assert b2._id == "custom"


def test_step_builder_required():
    client = Client()
    builder = client.new_step("Test").required("name", AttributeType.STRING)
    step = builder.build()
    assert "name" in step.attributes
    assert step.attributes["name"].role == AttributeRole.REQUIRED
    assert step.attributes["name"].type == AttributeType.STRING


def test_step_builder_optional():
    client = Client()
    builder = client.new_step("Test").optional(
        "count", AttributeType.NUMBER, "0"
    )
    step = builder.build()
    assert "count" in step.attributes
    assert step.attributes["count"].role == AttributeRole.OPTIONAL
    assert step.attributes["count"].default == "0"


def test_step_builder_output():
    client = Client()
    builder = client.new_step("Test").output("result", AttributeType.STRING)
    step = builder.build()
    assert "result" in step.attributes
    assert step.attributes["result"].role == AttributeRole.OUTPUT


def test_step_builder_with_for_each():
    client = Client()
    builder = (
        client.new_step("Test")
        .required("items", AttributeType.ARRAY)
        .with_for_each("items")
    )
    step = builder.build()
    assert step.attributes["items"].for_each is True


def test_step_builder_with_for_each_missing_attribute():
    client = Client()
    builder = client.new_step("Test")
    try:
        builder.with_for_each("nonexistent")
        assert False, "Should raise StepValidationError"
    except StepValidationError:
        pass


def test_step_builder_with_label():
    client = Client()
    builder = client.new_step("Test").with_label("env", "prod")
    step = builder.build()
    assert step.labels["env"] == "prod"


def test_step_builder_with_labels():
    client = Client()
    builder = client.new_step("Test").with_labels(
        {"env": "prod", "team": "platform"}
    )
    step = builder.build()
    assert step.labels["env"] == "prod"
    assert step.labels["team"] == "platform"


def test_step_builder_with_endpoint():
    client = Client()
    builder = client.new_step("Test").with_endpoint(
        "http://localhost:8081/test"
    )
    step = builder.build()
    assert step.http is not None
    assert step.http.endpoint == "http://localhost:8081/test"


def test_step_builder_with_health_check():
    client = Client()
    builder = (
        client.new_step("Test")
        .with_endpoint("http://localhost:8081/test")
        .with_health_check("http://localhost:8081/health")
    )
    step = builder.build()
    assert step.http is not None
    assert step.http.health_check == "http://localhost:8081/health"


def test_step_builder_with_timeout():
    client = Client()
    builder = (
        client.new_step("Test")
        .with_endpoint("http://localhost:8081/test")
        .with_timeout(5000)
    )
    step = builder.build()
    assert step.http is not None
    assert step.http.timeout_ms == 5000


def test_step_builder_with_script():
    client = Client()
    builder = client.new_step("Test").with_script("(+ 1 2)")
    step = builder.build()
    assert step.script is not None
    assert step.script.language == ScriptLanguage.ALE
    assert step.script.script == "(+ 1 2)"
    assert step.type == StepType.SCRIPT


def test_step_builder_with_script_language():
    client = Client()
    builder = client.new_step("Test").with_script_language(
        ScriptLanguage.LUA, "return 1 + 2"
    )
    step = builder.build()
    assert step.script is not None
    assert step.script.language == ScriptLanguage.LUA
    assert step.type == StepType.SCRIPT


def test_step_builder_with_predicate():
    client = Client()
    builder = client.new_step("Test").with_predicate(
        ScriptLanguage.ALE, "(> value 10)"
    )
    step = builder.build()
    assert step.predicate is not None
    assert step.predicate.script == "(> value 10)"


def test_step_builder_with_async_execution():
    client = Client()
    builder = client.new_step("Test").with_async_execution()
    step = builder.build()
    assert step.type == StepType.ASYNC


def test_step_builder_with_sync_execution():
    client = Client()
    builder = (
        client.new_step("Test").with_async_execution().with_sync_execution()
    )
    step = builder.build()
    assert step.type == StepType.SYNC


def test_step_builder_with_memoizable():
    client = Client()
    builder = client.new_step("Test").with_memoizable()
    step = builder.build()
    assert step.memoizable is True


def test_step_builder_chaining():
    client = Client()
    builder = (
        client.new_step("Test")
        .with_id("custom-id")
        .required("input", AttributeType.STRING)
        .output("output", AttributeType.STRING)
        .with_label("env", "test")
        .with_endpoint("http://localhost:8081/test")
    )
    step = builder.build()
    assert step.id == "custom-id"
    assert "input" in step.attributes
    assert "output" in step.attributes
    assert step.labels["env"] == "test"
    assert step.http.endpoint == "http://localhost:8081/test"


@responses.activate
def test_step_builder_register():
    responses.add(
        responses.POST,
        "http://localhost:8080/engine/step",
        json={},
        status=200,
    )

    client = Client()
    builder = client.new_step("Test").with_endpoint(
        "http://localhost:8081/test"
    )
    builder.register()

    assert len(responses.calls) == 1


def test_flow_builder_initialization():
    client = Client()
    builder = client.new_flow("flow-123")
    assert builder._flow_id == "flow-123"
    assert builder._goals == []


def test_flow_builder_with_goal():
    client = Client()
    builder = client.new_flow("flow-123").with_goal("step-1")
    assert builder._goals == ["step-1"]


def test_flow_builder_with_goals():
    client = Client()
    builder = client.new_flow("flow-123").with_goals("step-1", "step-2")
    assert builder._goals == ["step-1", "step-2"]


def test_flow_builder_with_initial_state():
    client = Client()
    builder = client.new_flow("flow-123").with_initial_state({"name": "Alice"})
    assert builder._initial_state == {"name": "Alice"}


@responses.activate
def test_flow_builder_start():
    responses.add(
        responses.POST,
        "http://localhost:8080/engine/flow",
        json={},
        status=200,
    )

    client = Client()
    builder = (
        client.new_flow("flow-123")
        .with_goals("step-1")
        .with_initial_state({"name": "Alice"})
    )
    builder.start()

    assert len(responses.calls) == 1
    req_body = responses.calls[0].request.body
    import json

    data = json.loads(req_body)
    assert data["flow_id"] == "flow-123"
    assert data["goals"] == ["step-1"]
    assert data["initial_state"] == {"name": "Alice"}


def test_kebab_case_conversion():
    from argyll.builder import _to_kebab_case

    assert _to_kebab_case("TestStep") == "test-step"
    assert _to_kebab_case("MyAwesomeStep") == "my-awesome-step"
    assert _to_kebab_case("test_step") == "test-step"
    assert _to_kebab_case("test step") == "test-step"
    assert _to_kebab_case("testStep") == "test-step"


def test_step_builder_build():
    client = Client()
    builder = (
        client.new_step("Test")
        .required("input", AttributeType.STRING)
        .output("output", AttributeType.STRING)
        .with_endpoint("http://localhost:8081/test")
    )
    step = builder.build()
    assert step.id == "test"
    assert step.name == "Test"
    assert step.type == StepType.SYNC


def test_step_builder_with_script_defaults_to_ale():
    client = Client()
    builder = client.new_step("ScriptStep").with_script("(+ 1 2)")
    step = builder.build()
    assert step.script is not None
    assert step.script.language == ScriptLanguage.ALE


def test_step_builder_const():
    client = Client()
    builder = client.new_step("Test").const(
        "api_key", AttributeType.STRING, '"secret"'
    )
    step = builder.build()
    assert "api_key" in step.attributes
    assert step.attributes["api_key"].role == AttributeRole.CONST
    assert step.attributes["api_key"].default == '"secret"'


def test_flow_builder_chaining():
    client = Client()
    builder = (
        client.new_flow("flow-123")
        .with_goal("step-1")
        .with_goal("step-2")
        .with_initial_state({"name": "Alice"})
    )
    assert builder._goals == ["step-1", "step-2"]
    assert builder._initial_state == {"name": "Alice"}


def test_step_builder_with_flow_goals():
    client = Client()
    builder = client.new_step("FlowStep").with_flow_goals("step-1", "step-2")
    step = builder.build()
    assert step.type == StepType.FLOW
    assert step.flow is not None
    assert step.flow.goals == ["step-1", "step-2"]


def test_step_builder_with_flow_input_map():
    client = Client()
    builder = (
        client.new_step("FlowStep")
        .with_flow_goals("step-1")
        .with_flow_input_map({"a": "b"})
    )
    step = builder.build()
    assert step.flow is not None
    assert step.flow.input_map == {"a": "b"}


def test_step_builder_with_flow_output_map():
    client = Client()
    builder = (
        client.new_step("FlowStep")
        .with_flow_goals("step-1")
        .with_flow_output_map({"c": "d"})
    )
    step = builder.build()
    assert step.flow is not None
    assert step.flow.output_map == {"c": "d"}


@responses.activate
def test_flow_builder_start_error():
    from argyll.errors import FlowError

    responses.add(
        responses.POST,
        "http://localhost:8080/engine/flow",
        json={"error": "Invalid flow"},
        status=400,
    )

    client = Client()
    builder = client.new_flow("flow-123").with_goals("step-1")

    try:
        builder.start()
        assert False, "Should have raised FlowError"
    except FlowError:
        pass
