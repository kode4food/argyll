"""Builder pattern for creating steps and flows."""

import copy
import re
from dataclasses import replace
from typing import TYPE_CHECKING, Any, Callable, Dict, List, Optional

from ._validation import validate_step
from .errors import StepRegistrationError, StepValidationError
from .types import (
    ActionMode,
    Args,
    AttributeRole,
    AttributeSpec,
    AttributeType,
    ConstConfig,
    FlowConfig,
    FlowID,
    Handling,
    HTTPAction,
    HTTPConfig,
    InitArgs,
    MetaConfig,
    OptionalConfig,
    RequiredConfig,
    ScriptConfig,
    Step,
    StepID,
    StepType,
    Tags,
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

    def __init__(self, client: "Client", name: str = "") -> None:
        self._client = client
        self._name = name
        self._id = _to_kebab_case(name) if name else ""
        self._type: StepType = StepType.SERVICE
        self._attributes: Dict[str, AttributeSpec] = {}
        self._description = ""
        self._tags: Tags = []
        self._http: Optional[HTTPConfig] = None
        self._script: Optional[ScriptConfig] = None
        self._predicate: Optional[ScriptConfig] = None
        self._flow: Optional[FlowConfig] = None
        self._compensate_handler: Optional[Callable[..., Any]] = None
        self._handling = Handling.STANDARD

    def _copy(self, **kwargs: Any) -> "StepBuilder":
        """Create a copy with updated attributes."""
        result = copy.copy(self)
        for key, value in kwargs.items():
            setattr(result, key, value)
        return result

    def with_id(self, step_id: StepID) -> "StepBuilder":
        """Set custom step ID."""
        return self._copy(_id=step_id)

    def with_name(self, name: str) -> "StepBuilder":
        """Set step name (auto-generates ID if unset)."""
        updates = {"_name": name}
        if not self._id and name:
            updates["_id"] = _to_kebab_case(name)
        return self._copy(**updates)

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
            role=AttributeRole.OPTIONAL,
            type=attr_type,
            optional=OptionalConfig(default=default),
        )
        return self._copy(_attributes=new_attrs)

    def const(
        self, name: str, attr_type: AttributeType, value: str
    ) -> "StepBuilder":
        """Add const input attribute with fixed value."""
        new_attrs = dict(self._attributes)
        new_attrs[name] = AttributeSpec(
            role=AttributeRole.CONST,
            type=attr_type,
            const=ConstConfig(value=value),
        )
        return self._copy(_attributes=new_attrs)

    def meta(self, name: str, meta_key: str) -> "StepBuilder":
        """Add meta input attribute drawn from execution metadata."""
        new_attrs = dict(self._attributes)
        new_attrs[name] = AttributeSpec(
            role=AttributeRole.META,
            type=AttributeType.ANY,
            meta=MetaConfig(key=meta_key),
        )
        return self._copy(_attributes=new_attrs)

    def output(self, name: str, attr_type: AttributeType) -> "StepBuilder":
        """Declare output attribute."""
        new_attrs = dict(self._attributes)
        new_attrs[name] = AttributeSpec(
            role=AttributeRole.OUTPUT, type=attr_type
        )
        return self._copy(_attributes=new_attrs)

    def with_for_each(self, *names: str) -> "StepBuilder":
        """Enable array iteration for attributes."""
        new_attrs = dict(self._attributes)
        for name in names:
            if name not in new_attrs:
                raise StepValidationError(f"Attribute {name} not defined")
            existing = new_attrs[name]
            if existing.role == AttributeRole.REQUIRED:
                rc = existing.required or RequiredConfig()
                new_attrs[name] = replace(
                    existing, required=replace(rc, for_each=True)
                )
            elif existing.role == AttributeRole.OPTIONAL:
                oc = existing.optional or OptionalConfig()
                new_attrs[name] = replace(
                    existing, optional=replace(oc, for_each=True)
                )
        return self._copy(_attributes=new_attrs)

    def with_description(self, description: str) -> "StepBuilder":
        """Set prose describing what the step does."""
        return self._copy(_description=description)

    def with_tags(self, *tags: str) -> "StepBuilder":
        """Merge tags into the step's tag set, sorted and deduped."""
        return self._copy(_tags=sorted({*self._tags, *tags}))

    def _http_with(self, **overrides: Any) -> "HTTPConfig":
        """Return a copy of the current HTTPConfig with the given overrides."""
        base = self._http
        invoke = base.invoke if base else HTTPAction(endpoint="")
        return HTTPConfig(
            invoke=HTTPAction(
                endpoint=overrides.get("endpoint", invoke.endpoint),
                method=overrides.get("method", invoke.method),
                timeout=overrides.get("timeout", invoke.timeout),
                mode=overrides.get("mode", invoke.mode),
            ),
            compensate=overrides.get(
                "compensate", base.compensate if base else None
            ),
            health=overrides.get("health", base.health if base else ""),
        )

    def _with_compensate(self, **overrides: Any) -> "StepBuilder":
        """Return a copy with updated compensation."""
        base = self._http.compensate if self._http else None
        action = HTTPAction(
            endpoint=overrides.get("endpoint", base.endpoint if base else ""),
            method=overrides.get("method", base.method if base else ""),
            timeout=overrides.get("timeout", base.timeout if base else 0),
            mode=overrides.get("mode", base.mode if base else ActionMode.SYNC),
        )
        return self._copy(
            _http=self._http_with(compensate=action),
            _handling=Handling.COMPENSATED,
        )

    def with_endpoint(self, url: str) -> "StepBuilder":
        """Set HTTP endpoint."""
        return self._copy(_http=self._http_with(endpoint=url))

    def with_method(self, method: str) -> "StepBuilder":
        """Set HTTP method."""
        return self._copy(_http=self._http_with(method=method.upper()))

    def with_health_check(self, url: str) -> "StepBuilder":
        """Set health check endpoint."""
        return self._copy(_http=self._http_with(health=url))

    def with_compensate(self, url: str) -> "StepBuilder":
        """Set the compensate endpoint and handling for the step."""
        return self._with_compensate(endpoint=url)

    def with_compensate_method(self, method: str) -> "StepBuilder":
        """Set the HTTP method used to compensate the step."""
        return self._with_compensate(method=method.upper())

    def with_compensate_timeout(self, ms: int) -> "StepBuilder":
        """Set the compensate timeout in milliseconds."""
        return self._with_compensate(timeout=ms)

    def with_timeout(self, ms: int) -> "StepBuilder":
        """Set execution timeout in milliseconds."""
        return self._copy(_http=self._http_with(timeout=ms))

    def with_script(self, script: ScriptConfig) -> "StepBuilder":
        """Set the script to execute."""
        return self._copy(_script=script, _type=StepType.SCRIPT)

    def with_predicate(self, predicate: ScriptConfig) -> "StepBuilder":
        """Set the predicate gating the step."""
        return self._copy(_predicate=predicate)

    def with_flow_goals(self, *goal_ids: StepID) -> "StepBuilder":
        """Configure flow step with goal IDs."""
        flow = self._flow or FlowConfig(goals=[])
        return self._copy(
            _flow=replace(flow, goals=list(goal_ids)),
            _type=StepType.FLOW,
        )

    def with_flow_space(self, space_id: str) -> "StepBuilder":
        """Restrict a child flow to a planning space."""
        flow = self._flow or FlowConfig(goals=[])
        return self._copy(
            _flow=replace(flow, space_id=space_id),
            _type=StepType.FLOW,
        )

    def with_invoke_mode(self, mode: ActionMode) -> "StepBuilder":
        """Set how the invoke action reports its result."""
        return self._copy(_http=self._http_with(mode=mode))

    def with_compensate_mode(self, mode: ActionMode) -> "StepBuilder":
        """Set how the compensate action reports its result."""
        return self._with_compensate(mode=mode)

    def with_async_execution(self) -> "StepBuilder":
        """Configure the invoke action to complete via webhook callback."""
        return self.with_invoke_mode(ActionMode.ASYNC)

    def with_sync_execution(self) -> "StepBuilder":
        """Configure the invoke action to complete in the HTTP response."""
        return self.with_invoke_mode(ActionMode.SYNC)

    def with_handling(self, handling: Handling) -> "StepBuilder":
        """Set how completed work is retained or reversed."""
        return self._copy(_handling=handling)

    def compensated(self, *names: str) -> "StepBuilder":
        """Include attributes in compensation requests."""
        new_attrs = dict(self._attributes)
        for name in names:
            if name not in new_attrs:
                raise StepValidationError(f"Attribute {name} not defined")
            new_attrs[name] = replace(new_attrs[name], compensated=True)
        return self._copy(_attributes=new_attrs)

    def with_compensate_handler(
        self, handler: Callable[..., Any]
    ) -> "StepBuilder":
        """Register a compensate handler: (ctx, arguments) -> None."""
        return self._copy(_compensate_handler=handler)

    def build(self) -> Step:
        """Create immutable Step."""
        if self._name and not self._id:
            self = self.with_name(self._name)
        step = Step(
            id=self._id,
            name=self._name,
            type=self._type,
            description=self._description,
            attributes=self._attributes,
            tags=self._tags,
            http=self._http,
            script=self._script,
            predicate=self._predicate,
            flow=self._flow,
            handling=self._handling,
        )
        validate_step(step)
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

    def start(self, handler: Callable[[Any, Args], Args]) -> None:
        """Build, register, and start Flask server."""
        from .handlers import create_step_server

        create_step_server(
            self._client,
            self,
            handler,
            compensate_handler=self._compensate_handler,
        )


class FlowBuilder:
    """Immutable builder for creating and starting flows."""

    def __init__(self, client: "Client", flow_id: FlowID) -> None:
        self._client = client
        self._flow_id = flow_id
        self._goals: List[StepID] = []
        self._initial_state: InitArgs = {}

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

    def with_initial_state(self, args: InitArgs) -> "FlowBuilder":
        """Set initial state."""
        return self._copy(_initial_state=args)

    def start(self) -> None:
        """Execute the flow."""
        from .errors import FlowError

        url = f"{self._client.base_url}/engine/flows"
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
