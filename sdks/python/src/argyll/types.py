"""Type definitions for the Argyll SDK."""

from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Dict, List, Optional

# Type aliases
Args = Dict[str, Any]
StepID = str
FlowID = str
Labels = Dict[str, str]
Metadata = Dict[str, Any]


class StepType(str, Enum):
    """Step execution type."""

    SYNC = "sync"
    ASYNC = "async"
    SCRIPT = "script"
    FLOW = "flow"


class AttributeRole(str, Enum):
    """Attribute role in step definition."""

    REQUIRED = "required"
    OPTIONAL = "optional"
    CONST = "const"
    OUTPUT = "output"


class AttributeType(str, Enum):
    """Attribute data type."""

    STRING = "string"
    NUMBER = "number"
    BOOLEAN = "boolean"
    OBJECT = "object"
    ARRAY = "array"
    NULL = "null"
    ANY = "any"


class ScriptLanguage(str, Enum):
    """Script language for script steps."""

    ALE = "ale"
    LUA = "lua"


class BackoffType(str, Enum):
    """Retry backoff strategy."""

    FIXED = "fixed"
    LINEAR = "linear"
    EXPONENTIAL = "exponential"


@dataclass(frozen=True)
class AttributeSpec:
    """Specification for a step attribute."""

    role: AttributeRole
    type: AttributeType
    default: str = ""
    for_each: bool = False

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        result: Dict[str, Any] = {
            "role": self.role.value,
            "type": self.type.value,
        }
        if self.default:
            result["default"] = self.default
        if self.for_each:
            result["for_each"] = True
        return result


@dataclass(frozen=True)
class HTTPConfig:
    """HTTP configuration for sync/async steps."""

    endpoint: str
    health_check: str = ""
    timeout_ms: int = 0

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        result: Dict[str, Any] = {"endpoint": self.endpoint}
        if self.health_check:
            result["health_check"] = self.health_check
        if self.timeout_ms > 0:
            result["timeout_ms"] = self.timeout_ms
        return result


@dataclass(frozen=True)
class ScriptConfig:
    """Script configuration for script steps."""

    language: ScriptLanguage
    script: str

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        return {"language": self.language.value, "script": self.script}


@dataclass(frozen=True)
class PredicateConfig:
    """Predicate configuration for conditional execution."""

    language: ScriptLanguage
    script: str

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        return {"language": self.language.value, "script": self.script}


@dataclass(frozen=True)
class RetryConfig:
    """Retry configuration for step execution."""

    max_attempts: int = 0
    backoff_type: BackoffType = BackoffType.FIXED
    backoff_ms: int = 0
    max_backoff_ms: int = 0

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        result: Dict[str, Any] = {}
        if self.max_attempts > 0:
            result["max_attempts"] = self.max_attempts
        if self.backoff_type:
            result["backoff_type"] = self.backoff_type.value
        if self.backoff_ms > 0:
            result["backoff_ms"] = self.backoff_ms
        if self.max_backoff_ms > 0:
            result["max_backoff_ms"] = self.max_backoff_ms
        return result


@dataclass(frozen=True)
class FlowConfig:
    """Flow configuration for flow steps."""

    goals: List[StepID]
    input_map: Dict[str, str] = field(default_factory=dict)
    output_map: Dict[str, str] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        result: Dict[str, Any] = {"goals": self.goals}
        if self.input_map:
            result["input_map"] = self.input_map
        if self.output_map:
            result["output_map"] = self.output_map
        return result


@dataclass(frozen=True)
class Step:
    """Complete step definition."""

    id: StepID
    name: str
    type: StepType
    attributes: Dict[str, AttributeSpec] = field(default_factory=dict)
    labels: Labels = field(default_factory=dict)
    http: Optional[HTTPConfig] = None
    script: Optional[ScriptConfig] = None
    predicate: Optional[PredicateConfig] = None
    retry: Optional[RetryConfig] = None
    flow: Optional[FlowConfig] = None
    memoizable: bool = False

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format with camelCase keys."""
        result: Dict[str, Any] = {
            "id": self.id,
            "name": self.name,
            "type": self.type.value,
        }

        if self.attributes:
            result["attributes"] = {
                k: v.to_dict() for k, v in self.attributes.items()
            }

        if self.labels:
            result["labels"] = self.labels

        if self.http:
            result["http"] = self.http.to_dict()

        if self.script:
            result["script"] = self.script.to_dict()

        if self.predicate:
            result["predicate"] = self.predicate.to_dict()

        if self.retry:
            result["retry"] = self.retry.to_dict()

        if self.flow:
            result["flow"] = self.flow.to_dict()

        if self.memoizable:
            result["memoizable"] = True

        return result


@dataclass(frozen=True)
class StepResult:
    """Result from step execution."""

    success: bool
    outputs: Args = field(default_factory=dict)
    error: str = ""

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        result: Dict[str, Any] = {"success": self.success}
        if self.outputs:
            result["outputs"] = self.outputs
        if self.error:
            result["error"] = self.error
        return result
