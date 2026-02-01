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
    FlowConfig,
    FlowID,
    HTTPConfig,
    Labels,
    Metadata,
    PredicateConfig,
    ScriptConfig,
    ScriptLanguage,
    Step,
    StepID,
    StepResult,
    StepType,
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
    "StepResult",
    "StepType",
    "AttributeRole",
    "AttributeType",
    "AttributeSpec",
    "ScriptLanguage",
    "BackoffType",
    "HTTPConfig",
    "ScriptConfig",
    "PredicateConfig",
    "FlowConfig",
    # Type aliases
    "Args",
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
