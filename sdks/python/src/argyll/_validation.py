"""Step validation logic."""

from ._attr_validation import (
    check_attribute_default,
    check_attribute_for_each,
    check_attribute_role_config,
)
from .errors import StepValidationError
from .types import (
    Step,
    StepType,
)


def validate_step(step: Step) -> None:
    """Validate a step definition, raising StepValidationError on failure."""
    _check_identity(step)
    _check_type_config(step)
    _check_attributes(step)


def _check_identity(step: Step) -> None:
    if not step.id:
        raise StepValidationError("Step ID cannot be empty")
    if not step.name:
        raise StepValidationError("Step name cannot be empty")
    if step.type not in {
        StepType.SYNC,
        StepType.ASYNC,
        StepType.SCRIPT,
        StepType.FLOW,
    }:
        raise StepValidationError(f"Invalid step type: {step.type}")


def _check_type_config(step: Step) -> None:
    if step.type in {StepType.SYNC, StepType.ASYNC}:
        if not step.http or not step.http.endpoint:
            raise StepValidationError("HTTP config with endpoint required")
        if step.http.method and step.http.method not in {
            "GET",
            "POST",
            "PUT",
            "DELETE",
        }:
            raise StepValidationError(
                f"Invalid HTTP method: {step.http.method}"
            )
        if step.flow is not None:
            raise StepValidationError("Flow config not allowed for HTTP steps")
        if step.script is not None:
            raise StepValidationError(
                "Script config not allowed for HTTP steps"
            )

    elif step.type == StepType.SCRIPT:
        if not step.script or not step.script.script:
            raise StepValidationError("Script config required for script step")
        if step.http is not None:
            raise StepValidationError(
                "HTTP config not allowed for script steps"
            )
        if step.flow is not None:
            raise StepValidationError(
                "Flow config not allowed for script steps"
            )

    elif step.type == StepType.FLOW:
        if not step.flow or not step.flow.goals:
            raise StepValidationError("Flow goals required for flow step")
        if step.http is not None:
            raise StepValidationError("HTTP config not allowed for flow steps")
        if step.script is not None:
            raise StepValidationError(
                "Script config not allowed for flow steps"
            )


def _check_attributes(step: Step) -> None:
    for name, spec in step.attributes.items():
        if not name:
            raise StepValidationError("Attribute name cannot be empty")
        check_attribute_role_config(name, spec)
        check_attribute_default(name, spec)
        check_attribute_for_each(name, spec)
