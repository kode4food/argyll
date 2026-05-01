"""Tests for type definitions."""

from argyll.types import (
    AttributeRole,
    AttributeSpec,
    AttributeType,
    ConstConfig,
    HTTPConfig,
    InputCollect,
    InputConfig,
    ProblemDetails,
    ScriptConfig,
    ScriptLanguage,
    Step,
    StepType,
)


def test_attribute_spec_to_dict():
    spec = AttributeSpec(role=AttributeRole.REQUIRED, type=AttributeType.STRING)
    result = spec.to_dict()
    assert result == {"role": "required", "type": "string"}


def test_attribute_spec_with_default():
    spec = AttributeSpec(
        role=AttributeRole.OPTIONAL,
        type=AttributeType.NUMBER,
        input=InputConfig(default="42"),
    )
    result = spec.to_dict()
    assert result == {
        "role": "optional",
        "type": "number",
        "input": {"default": "42"},
    }


def test_attribute_spec_with_for_each():
    spec = AttributeSpec(
        role=AttributeRole.REQUIRED,
        type=AttributeType.ARRAY,
        input=InputConfig(for_each=True),
    )
    result = spec.to_dict()
    assert result == {
        "role": "required",
        "type": "array",
        "input": {"for_each": True},
    }


def test_attribute_spec_with_collect():
    spec = AttributeSpec(
        role=AttributeRole.REQUIRED,
        type=AttributeType.ARRAY,
        input=InputConfig(collect=InputCollect.SOME),
    )
    result = spec.to_dict()
    assert result == {
        "role": "required",
        "type": "array",
        "input": {"collect": "some"},
    }


def test_attribute_spec_with_const():
    spec = AttributeSpec(
        role=AttributeRole.CONST,
        type=AttributeType.STRING,
        const=ConstConfig(value='"fixed"'),
    )
    result = spec.to_dict()
    assert result == {
        "role": "const",
        "type": "string",
        "const": {"value": '"fixed"'},
    }


def test_http_config_to_dict():
    config = HTTPConfig(endpoint="http://localhost:8081/step")
    result = config.to_dict()
    assert result == {"endpoint": "http://localhost:8081/step"}


def test_http_config_with_method():
    config = HTTPConfig(
        endpoint="http://localhost:8081/step",
        method="DELETE",
    )
    result = config.to_dict()
    assert result == {
        "endpoint": "http://localhost:8081/step",
        "method": "DELETE",
    }


def test_http_config_with_health_check():
    config = HTTPConfig(
        endpoint="http://localhost:8081/step",
        health_check="http://localhost:8081/health",
    )
    result = config.to_dict()
    assert result == {
        "endpoint": "http://localhost:8081/step",
        "health_check": "http://localhost:8081/health",
    }


def test_script_config_to_dict():
    config = ScriptConfig(language=ScriptLanguage.ALE, script="(+ 1 2)")
    result = config.to_dict()
    assert result == {"language": "ale", "script": "(+ 1 2)"}


def test_step_to_dict():
    step = Step(
        id="test-step",
        name="Test Step",
        type=StepType.SYNC,
        attributes={
            "input": AttributeSpec(
                role=AttributeRole.REQUIRED, type=AttributeType.STRING
            )
        },
        http=HTTPConfig(endpoint="http://localhost:8081/test"),
    )
    result = step.to_dict()
    assert result["id"] == "test-step"
    assert result["name"] == "Test Step"
    assert result["type"] == "sync"
    assert "input" in result["attributes"]
    assert result["http"]["endpoint"] == "http://localhost:8081/test"


def test_problem_details_to_dict():
    problem = ProblemDetails(status=422, detail="Something went wrong")
    result_dict = problem.to_dict()
    assert result_dict == {
        "type": "about:blank",
        "title": "Unprocessable Entity",
        "status": 422,
        "detail": "Something went wrong",
    }


def test_problem_details_with_title():
    problem = ProblemDetails(
        status=404, title="Not Found", detail="Missing resource"
    )
    result_dict = problem.to_dict()
    assert result_dict["title"] == "Not Found"
    assert result_dict["detail"] == "Missing resource"


def test_step_enums():
    assert StepType.SYNC.value == "sync"
    assert StepType.ASYNC.value == "async"
    assert StepType.SCRIPT.value == "script"
    assert AttributeRole.REQUIRED.value == "required"
    assert AttributeType.STRING.value == "string"
    assert ScriptLanguage.ALE.value == "ale"


def test_http_config_with_timeout():
    config = HTTPConfig(endpoint="http://localhost:8081/test", timeout=3000)
    result = config.to_dict()
    assert result["timeout"] == 3000


def test_work_config_to_dict():
    from argyll.types import BackoffType, WorkConfig

    config = WorkConfig(
        max_retries=5,
        backoff_type=BackoffType.EXPONENTIAL,
        backoff=100,
        max_backoff=5000,
    )
    result = config.to_dict()
    assert result["max_retries"] == 5
    assert result["backoff_type"] == "exponential"
    assert result["backoff"] == 100
    assert result["max_backoff"] == 5000


def test_flow_config_to_dict():
    from argyll.types import FlowConfig

    config = FlowConfig(goals=["step-1", "step-2"])
    result = config.to_dict()
    assert result["goals"] == ["step-1", "step-2"]


def test_predicate_config_to_dict():
    from argyll.types import PredicateConfig

    config = PredicateConfig(language=ScriptLanguage.LUA, script="return true")
    result = config.to_dict()
    assert result["language"] == "lua"
    assert result["script"] == "return true"


def test_step_with_all_fields():
    from argyll.types import (
        BackoffType,
        FlowConfig,
        PredicateConfig,
        WorkConfig,
    )

    step = Step(
        id="test-step",
        name="Test Step",
        type=StepType.ASYNC,
        attributes={
            "input": AttributeSpec(
                role=AttributeRole.REQUIRED, type=AttributeType.STRING
            )
        },
        labels={"env": "test"},
        http=HTTPConfig(
            endpoint="http://localhost:8081/test",
            method="POST",
            health_check="http://localhost:8081/health",
            timeout=5000,
        ),
        script=ScriptConfig(language=ScriptLanguage.ALE, script="(+ 1 2)"),
        predicate=PredicateConfig(
            language=ScriptLanguage.LUA, script="return true"
        ),
        work_config=WorkConfig(
            max_retries=3,
            backoff_type=BackoffType.LINEAR,
            backoff=1000,
            max_backoff=10000,
        ),
        flow=FlowConfig(goals=["step-1"]),
        memoizable=True,
    )

    result = step.to_dict()
    assert result["type"] == "async"
    assert result["labels"]["env"] == "test"
    assert result["http"]["method"] == "POST"
    assert result["http"]["timeout"] == 5000
    assert result["script"]["script"] == "(+ 1 2)"
    assert result["predicate"]["script"] == "return true"
    assert result["work_config"]["max_retries"] == 3
    assert result["flow"]["goals"] == ["step-1"]
    assert result["memoizable"] is True


def test_attribute_spec_no_optional_fields():
    spec = AttributeSpec(role=AttributeRole.OUTPUT, type=AttributeType.NUMBER)
    result = spec.to_dict()
    assert result == {"role": "output", "type": "number"}
    assert "input" not in result
    assert "const" not in result
