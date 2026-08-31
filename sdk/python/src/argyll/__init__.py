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
    ActionMode,
    Args,
    AttributeRole,
    AttributeSpec,
    AttributeType,
    BackoffType,
    ConstConfig,
    FlowConfig,
    FlowID,
    Handling,
    HTTPAction,
    HTTPConfig,
    InitArgs,
    InputCollect,
    MappingConfig,
    MetaConfig,
    Metadata,
    OptionalConfig,
    OutputConfig,
    ProblemDetails,
    RequiredConfig,
    ScriptConfig,
    ScriptLanguage,
    Step,
    StepID,
    StepType,
    Tags,
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
    "ActionMode",
    "Handling",
    "AttributeRole",
    "AttributeType",
    "AttributeSpec",
    "InputCollect",
    "RequiredConfig",
    "OptionalConfig",
    "ConstConfig",
    "MetaConfig",
    "OutputConfig",
    "MappingConfig",
    "ScriptLanguage",
    "BackoffType",
    "HTTPAction",
    "HTTPConfig",
    "ScriptConfig",
    "FlowConfig",
    "WorkConfig",
    # Type aliases
    "Args",
    "InitArgs",
    "StepID",
    "FlowID",
    "Tags",
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
