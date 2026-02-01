"""Builder pattern for creating steps and flows."""

import copy
import re
from typing import TYPE_CHECKING, Any, Callable, Dict, List, Optional

from .errors import StepRegistrationError, StepValidationError
from .types import (
    Args,
    AttributeRole,
    AttributeSpec,
    AttributeType,
    FlowConfig,
    FlowID,
    HTTPConfig,
    Labels,
    PredicateConfig,
    ScriptConfig,
    ScriptLanguage,
    Step,
    StepID,
    StepResult,
    StepType,
)

if TYPE_CHECKING:
    from .client import Client


def _to_kebab_case(s: str) -> str:
    """Convert string to kebab-case."""
    s = re.sub(r"([a-z0-9])([A-Z])", r"\1-\2", s)
    s = re.sub(r"[\s_]+", "-", s)
    return s.lower()


class StepBuilder:
    """Immutable builder for creating step definitions."""

    def __init__(
        self,
        client: "Client",
        name: str,
        step_id: Optional[StepID] = None,
        step_type: StepType = StepType.SYNC,
        attributes: Optional[Dict[str, AttributeSpec]] = None,
        labels: Optional[Labels] = None,
        http: Optional[HTTPConfig] = None,
        script: Optional[ScriptConfig] = None,
        predicate: Optional[PredicateConfig] = None,
        flow: Optional[FlowConfig] = None,
        memoizable: bool = False,
    ) -> None:
        self._client = client
        self._name = name
        self._id = step_id or _to_kebab_case(name)
        self._type = step_type
        self._attributes = attributes or {}
        self._labels = labels or {}
        self._http = http
        self._script = script
        self._predicate = predicate
        self._flow = flow
        self._memoizable = memoizable

    def _copy(self, **kwargs: Any) -> "StepBuilder":
        """Create a copy with updated attributes."""
        result = copy.copy(self)
        for key, value in kwargs.items():
            setattr(result, key, value)
        return result

    def with_id(self, step_id: StepID) -> "StepBuilder":
        """Set custom step ID."""
        return self._copy(_id=step_id)

    def required(self, name: str, attr_type: AttributeType) -> "StepBuilder":
        """Add required input attribute."""
        new_attrs = dict(self._attributes)
        new_attrs[name] = AttributeSpec(
            role=AttributeRole.REQUIRED, type=attr_type
        )
        return self._copy(_attributes=new_attrs)

    def optional(
        self, name: str, attr_type: AttributeType, default: str
    ) -> "StepBuilder":
        """Add optional input attribute with default value."""
        new_attrs = dict(self._attributes)
        new_attrs[name] = AttributeSpec(
            role=AttributeRole.OPTIONAL, type=attr_type, default=default
        )
        return self._copy(_attributes=new_attrs)

    def const(
        self, name: str, attr_type: AttributeType, value: str
    ) -> "StepBuilder":
        """Add const input attribute with fixed value."""
        new_attrs = dict(self._attributes)
        new_attrs[name] = AttributeSpec(
            role=AttributeRole.CONST, type=attr_type, default=value
        )
        return self._copy(_attributes=new_attrs)

    def output(self, name: str, attr_type: AttributeType) -> "StepBuilder":
        """Declare output attribute."""
        new_attrs = dict(self._attributes)
        new_attrs[name] = AttributeSpec(
            role=AttributeRole.OUTPUT, type=attr_type
        )
        return self._copy(_attributes=new_attrs)

    def with_for_each(self, name: str) -> "StepBuilder":
        """Enable array iteration for attribute."""
        if name not in self._attributes:
            raise StepValidationError(f"Attribute {name} not defined")

        new_attrs = dict(self._attributes)
        existing = new_attrs[name]
        new_attrs[name] = AttributeSpec(
            role=existing.role,
            type=existing.type,
            default=existing.default,
            for_each=True,
        )
        return self._copy(_attributes=new_attrs)

    def with_label(self, key: str, value: str) -> "StepBuilder":
        """Add single label."""
        new_labels = dict(self._labels)
        new_labels[key] = value
        return self._copy(_labels=new_labels)

    def with_labels(self, labels: Labels) -> "StepBuilder":
        """Merge labels."""
        new_labels = dict(self._labels)
        new_labels.update(labels)
        return self._copy(_labels=new_labels)

    def with_endpoint(self, url: str) -> "StepBuilder":
        """Set HTTP endpoint."""
        new_http = HTTPConfig(
            endpoint=url,
            health_check=self._http.health_check if self._http else "",
            timeout_ms=self._http.timeout_ms if self._http else 0,
        )
        return self._copy(_http=new_http)

    def with_health_check(self, url: str) -> "StepBuilder":
        """Set health check endpoint."""
        new_http = HTTPConfig(
            endpoint=self._http.endpoint if self._http else "",
            health_check=url,
            timeout_ms=self._http.timeout_ms if self._http else 0,
        )
        return self._copy(_http=new_http)

    def with_timeout(self, ms: int) -> "StepBuilder":
        """Set execution timeout in milliseconds."""
        new_http = HTTPConfig(
            endpoint=self._http.endpoint if self._http else "",
            health_check=self._http.health_check if self._http else "",
            timeout_ms=ms,
        )
        return self._copy(_http=new_http)

    def with_script(self, script: str) -> "StepBuilder":
        """Set Ale script."""
        return self.with_script_language(ScriptLanguage.ALE, script)

    def with_script_language(
        self, language: ScriptLanguage, script: str
    ) -> "StepBuilder":
        """Set script with specific language."""
        new_script = ScriptConfig(language=language, script=script)
        return self._copy(_script=new_script, _type=StepType.SCRIPT)

    def with_predicate(
        self, language: ScriptLanguage, script: str
    ) -> "StepBuilder":
        """Set predicate for conditional execution."""
        new_predicate = PredicateConfig(language=language, script=script)
        return self._copy(_predicate=new_predicate)

    def with_flow_goals(self, *goal_ids: StepID) -> "StepBuilder":
        """Configure flow step with goal IDs."""
        new_flow = FlowConfig(
            goals=list(goal_ids),
            input_map=self._flow.input_map if self._flow else {},
            output_map=self._flow.output_map if self._flow else {},
        )
        return self._copy(_flow=new_flow, _type=StepType.FLOW)

    def with_flow_input_map(self, mapping: Dict[str, str]) -> "StepBuilder":
        """Configure input mapping for flow step."""
        new_flow = FlowConfig(
            goals=self._flow.goals if self._flow else [],
            input_map=mapping,
            output_map=self._flow.output_map if self._flow else {},
        )
        return self._copy(_flow=new_flow, _type=StepType.FLOW)

    def with_flow_output_map(self, mapping: Dict[str, str]) -> "StepBuilder":
        """Configure output mapping for flow step."""
        new_flow = FlowConfig(
            goals=self._flow.goals if self._flow else [],
            input_map=self._flow.input_map if self._flow else {},
            output_map=mapping,
        )
        return self._copy(_flow=new_flow, _type=StepType.FLOW)

    def with_async_execution(self) -> "StepBuilder":
        """Configure async execution."""
        return self._copy(_type=StepType.ASYNC)

    def with_sync_execution(self) -> "StepBuilder":
        """Configure sync execution."""
        return self._copy(_type=StepType.SYNC)

    def with_memoizable(self) -> "StepBuilder":
        """Enable result memoization."""
        return self._copy(_memoizable=True)

    def build(self) -> Step:
        """Create immutable Step."""
        return Step(
            id=self._id,
            name=self._name,
            type=self._type,
            attributes=self._attributes,
            labels=self._labels,
            http=self._http,
            script=self._script,
            predicate=self._predicate,
            flow=self._flow,
            memoizable=self._memoizable,
        )

    def register(self) -> None:
        """Build and register step with engine."""
        step = self.build()
        try:
            self._client.register_step(step)
        except Exception as e:
            raise StepRegistrationError(
                f"Failed to register step {step.id}: {e}"
            ) from e

    def start(self, handler: Callable[[Any, Args], StepResult]) -> None:
        """Build, register, and start Flask server."""
        from .handlers import create_step_server

        create_step_server(self._client, self, handler)


class FlowBuilder:
    """Immutable builder for creating and starting flows."""

    def __init__(
        self,
        client: "Client",
        flow_id: FlowID,
        goals: Optional[List[StepID]] = None,
        initial_state: Optional[Args] = None,
    ) -> None:
        self._client = client
        self._flow_id = flow_id
        self._goals = goals or []
        self._initial_state = initial_state or {}

    def _copy(self, **kwargs: Any) -> "FlowBuilder":
        """Create a copy with updated attributes."""
        result = copy.copy(self)
        for key, value in kwargs.items():
            setattr(result, key, value)
        return result

    def with_goal(self, step_id: StepID) -> "FlowBuilder":
        """Add single goal step."""
        new_goals = list(self._goals)
        new_goals.append(step_id)
        return self._copy(_goals=new_goals)

    def with_goals(self, *step_ids: StepID) -> "FlowBuilder":
        """Set all goal steps."""
        return self._copy(_goals=list(step_ids))

    def with_initial_state(self, args: Args) -> "FlowBuilder":
        """Set initial state."""
        return self._copy(_initial_state=args)

    def start(self) -> None:
        """Execute the flow."""
        from .errors import FlowError

        url = f"{self._client.base_url}/flow"
        payload = {
            "flow_id": self._flow_id,
            "goals": self._goals,
            "initial_state": self._initial_state,
        }

        try:
            resp = self._client.session.post(
                url, json=payload, timeout=self._client.timeout
            )
            resp.raise_for_status()
        except Exception as e:
            raise FlowError(f"Failed to start flow {self._flow_id}: {e}") from e
