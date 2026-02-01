"""Builder pattern for creating steps and flows."""

import copy
import json
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
        dirty: bool = False,
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
        self._dirty = dirty

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
            timeout=self._http.timeout if self._http else 0,
        )
        return self._copy(_http=new_http)

    def with_health_check(self, url: str) -> "StepBuilder":
        """Set health check endpoint."""
        new_http = HTTPConfig(
            endpoint=self._http.endpoint if self._http else "",
            health_check=url,
            timeout=self._http.timeout if self._http else 0,
        )
        return self._copy(_http=new_http)

    def with_timeout(self, ms: int) -> "StepBuilder":
        """Set execution timeout in milliseconds."""
        new_http = HTTPConfig(
            endpoint=self._http.endpoint if self._http else "",
            health_check=self._http.health_check if self._http else "",
            timeout=ms,
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

    def update(self) -> "StepBuilder":
        """Mark step as updated (uses update instead of register on start)."""
        return self._copy(_dirty=True)

    def build(self) -> Step:
        """Create immutable Step."""
        step = Step(
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
        _validate_step(step)
        return step

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

        url = f"{self._client.base_url}/engine/flow"
        payload = {
            "id": self._flow_id,
            "goals": self._goals,
            "init": self._initial_state,
        }

        try:
            resp = self._client.session.post(
                url, json=payload, timeout=self._client.timeout
            )
            resp.raise_for_status()
        except Exception as e:
            raise FlowError(f"Failed to start flow {self._flow_id}: {e}") from e


def _validate_step(step: Step) -> None:
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

    if step.type in {StepType.SYNC, StepType.ASYNC}:
        if not step.http or not step.http.endpoint:
            raise StepValidationError("HTTP config with endpoint required")
        if step.flow is not None:
            raise StepValidationError("Flow config not allowed for HTTP steps")
        if step.script is not None:
            raise StepValidationError("Script config not allowed for HTTP steps")

    if step.type == StepType.SCRIPT:
        if not step.script or not step.script.script:
            raise StepValidationError("Script config required for script step")
        if step.http is not None:
            raise StepValidationError("HTTP config not allowed for script steps")
        if step.flow is not None:
            raise StepValidationError("Flow config not allowed for script steps")

    if step.type == StepType.FLOW:
        if not step.flow or not step.flow.goals:
            raise StepValidationError("Flow goals required for flow step")
        if step.http is not None:
            raise StepValidationError("HTTP config not allowed for flow steps")
        if step.script is not None:
            raise StepValidationError("Script config not allowed for flow steps")

    for name, spec in step.attributes.items():
        if not name:
            raise StepValidationError("Attribute name cannot be empty")

        if spec.role == AttributeRole.CONST and not spec.default:
            raise StepValidationError(
                f"Const attribute {name} requires default value"
            )

        if spec.default and spec.role not in {
            AttributeRole.OPTIONAL,
            AttributeRole.CONST,
        }:
            raise StepValidationError(
                f"Default value not allowed for attribute {name}"
            )

        if spec.default:
            try:
                parsed = json.loads(spec.default)
            except json.JSONDecodeError as e:
                raise StepValidationError(
                    f"Invalid default JSON for {name}: {e}"
                ) from e

            if spec.type == AttributeType.STRING and not isinstance(
                parsed, str
            ):
                raise StepValidationError(
                    f"Default for {name} must be JSON string"
                )
            if spec.type == AttributeType.NUMBER and not isinstance(
                parsed, (int, float)
            ):
                raise StepValidationError(
                    f"Default for {name} must be JSON number"
                )
            if spec.type == AttributeType.BOOLEAN and not isinstance(
                parsed, bool
            ):
                raise StepValidationError(
                    f"Default for {name} must be JSON boolean"
                )
            if spec.type == AttributeType.OBJECT and not isinstance(
                parsed, dict
            ):
                raise StepValidationError(
                    f"Default for {name} must be JSON object"
                )
            if spec.type == AttributeType.ARRAY and not isinstance(
                parsed, list
            ):
                raise StepValidationError(
                    f"Default for {name} must be JSON array"
                )
            if spec.type == AttributeType.NULL and parsed is not None:
                raise StepValidationError(
                    f"Default for {name} must be JSON null"
                )

        if spec.for_each:
            if spec.role == AttributeRole.OUTPUT:
                raise StepValidationError(
                    f"ForEach not allowed for output attribute {name}"
                )
            if spec.type not in {AttributeType.ARRAY, AttributeType.ANY}:
                raise StepValidationError(
                    f"ForEach requires array/any type for {name}"
                )
