"""Attribute-level validation logic."""

import json
from typing import Union

from .errors import StepValidationError
from .types import (
    AttributeRole,
    AttributeSpec,
    AttributeType,
)

_TYPE_CHECKS: dict[AttributeType, tuple[Union[type, tuple[type, ...]], str]] = {
    AttributeType.STRING: (str, "JSON string"),
    AttributeType.NUMBER: ((int, float), "JSON number"),
    AttributeType.BOOLEAN: (bool, "JSON boolean"),
    AttributeType.OBJECT: (dict, "JSON object"),
    AttributeType.ARRAY: (list, "JSON array"),
}

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
        parsed = json.loads(default)
    except json.JSONDecodeError as e:
        raise StepValidationError(
            f"Invalid default JSON for {name}: {e}"
        ) from e

    if spec.type in _TYPE_CHECKS:
        expected_type, label = _TYPE_CHECKS[spec.type]
        if not isinstance(parsed, expected_type):
            raise StepValidationError(f"Default for {name} must be {label}")
    elif spec.type == AttributeType.NULL and parsed is not None:
        raise StepValidationError(f"Default for {name} must be JSON null")


def check_attribute_for_each(name: str, spec: AttributeSpec) -> None:
    for_each = False
    if spec.role == AttributeRole.REQUIRED and spec.required:
        for_each = spec.required.for_each
    elif spec.role == AttributeRole.OPTIONAL and spec.optional:
        for_each = spec.optional.for_each

    if for_each and spec.type not in {AttributeType.ARRAY, AttributeType.ANY}:
        raise StepValidationError(f"ForEach requires array/any type for {name}")
