"""Step validation logic."""

from ._attr_validation import (
    check_attribute_default,
    check_attribute_for_each,
    check_attribute_role_config,
)
from .errors import StepValidationError
from .types import (
    AttributeSpec,
    Handling,
    Step,
    StepType,
)


def validate_step(step: Step) -> None:
    """Validate a step definition, raising StepValidationError on failure."""
    _check_identity(step)
    _check_type_config(step)
    _check_attributes(step)
    _check_compensation(step)


def _check_identity(step: Step) -> None:
    if not step.id:
        raise StepValidationError("Step ID cannot be empty")
    if not step.name:
        raise StepValidationError("Step name cannot be empty")
    if step.type not in {
        StepType.SERVICE,
        StepType.SCRIPT,
        StepType.FLOW,
    }:
        raise StepValidationError(f"Invalid step type: {step.type}")
    if step.handling not in {
        Handling.STANDARD,
        Handling.MEMOIZED,
        Handling.COMPENSATED,
    }:
        raise StepValidationError(f"Invalid step handling: {step.handling}")


def _check_type_config(step: Step) -> None:
    if step.type == StepType.SERVICE:
        if not step.http or not step.http.invoke.endpoint:
            raise StepValidationError("HTTP config with endpoint required")
        for action in (step.http.invoke, step.http.compensate):
            if action is None or not action.method:
                continue
            if action.method not in {"GET", "POST", "PUT", "DELETE"}:
                raise StepValidationError(
                    f"Invalid HTTP method: {action.method}"
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


def _check_compensation(step: Step) -> None:
    action = step.http.compensate if step.http else None
    if step.handling == Handling.COMPENSATED and (
        action is None or not action.endpoint
    ):
        raise StepValidationError(
            "Compensated handling requires a compensation endpoint"
        )
    if step.handling != Handling.COMPENSATED and action is not None:
        raise StepValidationError(
            "Compensation endpoint requires compensated handling"
        )

    names = set()
    for name, spec in step.attributes.items():
        if not spec.compensated:
            continue
        if step.handling != Handling.COMPENSATED:
            raise StepValidationError(
                f"Compensated attribute requires compensated handling: {name}"
            )
        mapped = _mapped_name(name, spec)
        if mapped in names:
            raise StepValidationError(
                f"Conflicting compensation argument: {mapped}"
            )
        names.add(mapped)


def _mapped_name(name: str, spec: AttributeSpec) -> str:
    config = spec.output or spec.required or spec.optional
    if config and config.mapping and config.mapping.name:
        return config.mapping.name
    return name
