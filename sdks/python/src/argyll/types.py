"""Type definitions for the Argyll SDK."""

from dataclasses import dataclass, field
from enum import Enum
from http import HTTPStatus
from typing import Any, Dict, List, Optional

# Type aliases
Args = Dict[str, Any]
InitArgs = Dict[str, List[Any]]
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


class InputCollect(str, Enum):
    """Input collection mode."""

    FIRST = "first"
    LAST = "last"
    ALL = "all"
    SOME = "some"
    NONE = "none"


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
class InputConfig:
    """Input collection configuration."""

    collect: InputCollect = InputCollect.FIRST
    default: str = ""
    for_each: bool = False
    timeout: int = 0

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        result: Dict[str, Any] = {}
        if self.collect != InputCollect.FIRST:
            result["collect"] = self.collect.value
        if self.default:
            result["default"] = self.default
        if self.for_each:
            result["for_each"] = True
        if self.timeout > 0:
            result["timeout"] = self.timeout
        return result


@dataclass(frozen=True)
class ConstConfig:
    """Const input configuration."""

    value: str

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        return {"value": self.value}


@dataclass(frozen=True)
class AttributeSpec:
    """Specification for a step attribute."""

    role: AttributeRole
    type: AttributeType
    input: Optional[InputConfig] = None
    const: Optional[ConstConfig] = None

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        result: Dict[str, Any] = {
            "role": self.role.value,
            "type": self.type.value,
        }
        if self.input is not None:
            result["input"] = self.input.to_dict()
        if self.const is not None:
            result["const"] = self.const.to_dict()
        return result


@dataclass(frozen=True)
class HTTPConfig:
    """HTTP configuration for sync/async steps."""

    endpoint: str
    method: str = ""
    health_check: str = ""
    timeout: int = 0

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        result: Dict[str, Any] = {"endpoint": self.endpoint}
        if self.method:
            result["method"] = self.method
        if self.health_check:
            result["health_check"] = self.health_check
        if self.timeout > 0:
            result["timeout"] = self.timeout
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
class WorkConfig:
    """Work configuration for retries and parallelism."""

    max_retries: int = 0
    backoff_type: BackoffType = BackoffType.FIXED
    backoff: int = 0
    max_backoff: int = 0
    parallelism: int = 0

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        result: Dict[str, Any] = {}
        if self.max_retries > 0:
            result["max_retries"] = self.max_retries
        if self.backoff_type:
            result["backoff_type"] = self.backoff_type.value
        if self.backoff > 0:
            result["backoff"] = self.backoff
        if self.max_backoff > 0:
            result["max_backoff"] = self.max_backoff
        if self.parallelism > 0:
            result["parallelism"] = self.parallelism
        return result


@dataclass(frozen=True)
class FlowConfig:
    """Flow configuration for flow steps."""

    goals: List[StepID]

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        return {"goals": self.goals}


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
    work_config: Optional[WorkConfig] = None
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

        if self.work_config:
            result["work_config"] = self.work_config.to_dict()

        if self.flow:
            result["flow"] = self.flow.to_dict()

        if self.memoizable:
            result["memoizable"] = True

        return result


@dataclass(frozen=True)
class ProblemDetails:
    """RFC 9457 problem details for failed step execution."""

    status: int
    detail: str
    type: str = "about:blank"
    title: str = ""
    instance: str = ""

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API dictionary format."""
        result: Dict[str, Any] = {
            "type": self.type,
            "title": self.title or _status_title(self.status),
            "status": self.status,
            "detail": self.detail,
        }
        if self.instance:
            result["instance"] = self.instance
        return result


def _status_title(status: int) -> str:
    if status == 422:
        return "Unprocessable Entity"
    try:
        return HTTPStatus(status).phrase
    except ValueError:
        return ""
