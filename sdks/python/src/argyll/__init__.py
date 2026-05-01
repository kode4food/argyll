"""Argyll SDK for Python."""

from .builder import FlowBuilder, StepBuilder
from .client import Client, FlowClient
from .errors import (
    ArgyllError,
    ClientError,
    FlowError,
    HTTPError,
    StepRegistrationError,
    StepValidationError,
    WebhookError,
)
from .handlers import AsyncContext, StepContext, StepHandler
from .types import (
    Args,
    AttributeRole,
    AttributeSpec,
    AttributeType,
    BackoffType,
    ConstConfig,
    FlowConfig,
    FlowID,
    HTTPConfig,
    InitArgs,
    InputCollect,
    InputConfig,
    Labels,
    Metadata,
    PredicateConfig,
    ProblemDetails,
    ScriptConfig,
    ScriptLanguage,
    Step,
    StepID,
    StepType,
    WorkConfig,
)

__version__ = "0.1.0"

__all__ = [
    # Client
    "Client",
    "FlowClient",
    # Builders
    "StepBuilder",
    "FlowBuilder",
    # Handlers
    "StepContext",
    "AsyncContext",
    "StepHandler",
    # Types
    "Step",
    "ProblemDetails",
    "StepType",
    "AttributeRole",
    "AttributeType",
    "AttributeSpec",
    "InputCollect",
    "InputConfig",
    "ConstConfig",
    "ScriptLanguage",
    "BackoffType",
    "HTTPConfig",
    "ScriptConfig",
    "PredicateConfig",
    "FlowConfig",
    "WorkConfig",
    # Type aliases
    "Args",
    "InitArgs",
    "StepID",
    "FlowID",
    "Labels",
    "Metadata",
    # Errors
    "ArgyllError",
    "ClientError",
    "StepRegistrationError",
    "StepValidationError",
    "FlowError",
    "WebhookError",
    "HTTPError",
]
