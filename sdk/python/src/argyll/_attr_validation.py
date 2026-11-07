"""Attribute-level validation logic."""

import json

from .errors import StepValidationError
from .types import AttributeRole, AttributeSpec

_ROLE_EXCLUSIVE = {
    AttributeRole.REQUIRED: ("optional", "const", "meta", "output"),
    AttributeRole.OPTIONAL: ("required", "const", "meta", "output"),
    AttributeRole.CONST: ("required", "optional", "meta", "output"),
    AttributeRole.META: ("required", "optional", "const", "output"),
    AttributeRole.OUTPUT: ("required", "optional", "const", "meta"),
}


def check_attribute_role_config(name: str, spec: AttributeSpec) -> None:
    forbidden = _ROLE_EXCLUSIVE.get(spec.role, ())
    if any(getattr(spec, f) is not None for f in forbidden):
        raise StepValidationError(
            f"Wrong config type for {spec.role.value} attribute {name}"
        )
    if spec.role == AttributeRole.CONST and (
        not spec.const or not spec.const.value
    ):
        raise StepValidationError(
            f"Const attribute {name} requires const value"
        )
    if spec.role == AttributeRole.META and (not spec.meta or not spec.meta.key):
        raise StepValidationError(f"Meta attribute {name} requires meta key")


def check_attribute_default(name: str, spec: AttributeSpec) -> None:
    default = ""
    if spec.role == AttributeRole.OPTIONAL and spec.optional:
        default = spec.optional.default
    elif spec.role == AttributeRole.CONST and spec.const:
        default = spec.const.value

    if not default:
        return

    try:
        json.loads(default)
    except json.JSONDecodeError as e:
        raise StepValidationError(
            f"Invalid default JSON for {name}: {e}"
        ) from e
