"""HTTP client for Argyll engine API."""

from typing import TYPE_CHECKING, Any, Dict, List, Optional

import requests

from .errors import ClientError, FlowError
from .types import FlowID, Step

if TYPE_CHECKING:
    from .builder import FlowBuilder, StepBuilder


class FlowClient:
    """Client for flow-scoped operations."""

    def __init__(self, client: "Client", flow_id: FlowID) -> None:
        self._client = client
        self._flow_id = flow_id

    @property
    def flow_id(self) -> FlowID:
        """Get the flow ID."""
        return self._flow_id

    def get_state(self) -> Dict[str, Any]:
        """Get the current flow state."""
        url = f"{self._client.base_url}/engine/flow/{self._flow_id}"
        try:
            resp = self._client.session.get(url, timeout=self._client.timeout)
            resp.raise_for_status()
            data: Dict[str, Any] = resp.json()
            return data
        except requests.RequestException as e:
            raise FlowError(f"Failed to get flow state: {e}") from e


class Client:
    """HTTP client for Argyll engine API."""

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        timeout: int = 30,
        session: Optional[requests.Session] = None,
    ) -> None:
        trimmed = base_url.rstrip("/")
        if trimmed.endswith("/engine"):
            trimmed = trimmed[: -len("/engine")]
        self.base_url = trimmed
        self.timeout = timeout
        self.session = session or requests.Session()

    def list_steps(self) -> List[Step]:
        """List all registered steps."""
        url = f"{self.base_url}/engine/step"
        try:
            resp = self.session.get(url, timeout=self.timeout)
            resp.raise_for_status()
            data = resp.json()
            if isinstance(data, dict):
                step_items = data.get("steps", [])
            else:
                step_items = data

            steps = []
            for step_data in step_items:
                steps.append(self._parse_step(step_data))
            return steps
        except requests.RequestException as e:
            status = getattr(e.response, "status_code", None)
            raise ClientError(
                f"Failed to list steps: {e}", status_code=status
            ) from e

    def register_step(self, step: Step) -> None:
        """Register a new step with the engine."""
        url = f"{self.base_url}/engine/step"
        try:
            resp = self.session.post(
                url, json=step.to_dict(), timeout=self.timeout
            )
            resp.raise_for_status()
        except requests.RequestException as e:
            status = getattr(e.response, "status_code", None)
            msg = f"Failed to register step {step.id}: {e}"
            raise ClientError(msg, status_code=status) from e

    def update_step(self, step: Step) -> None:
        """Update an existing step."""
        url = f"{self.base_url}/engine/step/{step.id}"
        try:
            resp = self.session.put(
                url, json=step.to_dict(), timeout=self.timeout
            )
            resp.raise_for_status()
        except requests.RequestException as e:
            status = getattr(e.response, "status_code", None)
            msg = f"Failed to update step {step.id}: {e}"
            raise ClientError(msg, status_code=status) from e

    def new_step(self, name: str = "") -> "StepBuilder":
        """Create a new step builder."""
        from .builder import StepBuilder

        return StepBuilder(client=self, name=name)

    def new_flow(self, flow_id: FlowID) -> "FlowBuilder":
        """Create a new flow builder."""
        from .builder import FlowBuilder

        return FlowBuilder(client=self, flow_id=flow_id)

    def flow(self, flow_id: FlowID) -> FlowClient:
        """Get a flow client for the specified flow."""
        return FlowClient(self, flow_id)

    def _parse_step(self, data: Dict[str, Any]) -> Step:
        """Parse step data from API response."""
        from .types import (
            AttributeRole,
            AttributeSpec,
            AttributeType,
            BackoffType,
            FlowConfig,
            HTTPConfig,
            PredicateConfig,
            ScriptConfig,
            ScriptLanguage,
            StepType,
            WorkConfig,
        )

        # Parse attributes
        attributes = {}
        for name, spec_data in data.get("attributes", {}).items():
            attributes[name] = AttributeSpec(
                role=AttributeRole(spec_data["role"]),
                type=AttributeType(spec_data["type"]),
                default=spec_data.get("default", ""),
                for_each=spec_data.get("for_each", False),
            )

        # Parse HTTP config
        http = None
        if "http" in data:
            http_data = data["http"]
            http = HTTPConfig(
                endpoint=http_data["endpoint"],
                health_check=http_data.get("health_check", ""),
                timeout=http_data.get("timeout", 0),
            )

        # Parse script config
        script = None
        if "script" in data:
            script_data = data["script"]
            script = ScriptConfig(
                language=ScriptLanguage(script_data["language"]),
                script=script_data["script"],
            )

        # Parse predicate config
        predicate = None
        if "predicate" in data:
            pred_data = data["predicate"]
            predicate = PredicateConfig(
                language=ScriptLanguage(pred_data["language"]),
                script=pred_data["script"],
            )

        # Parse work config
        work_config = None
        if "work_config" in data:
            work_data = data["work_config"]
            work_config = WorkConfig(
                max_retries=work_data.get("max_retries", 0),
                backoff_type=BackoffType(
                    work_data.get("backoff_type", "fixed")
                ),
                backoff=work_data.get("backoff", 0),
                max_backoff=work_data.get("max_backoff", 0),
                parallelism=work_data.get("parallelism", 0),
            )

        # Parse flow config
        flow = None
        if "flow" in data:
            flow_data = data["flow"]
            flow = FlowConfig(goals=flow_data["goals"])

        return Step(
            id=data["id"],
            name=data["name"],
            type=StepType(data["type"]),
            attributes=attributes,
            labels=data.get("labels", {}),
            http=http,
            script=script,
            predicate=predicate,
            work_config=work_config,
            flow=flow,
            memoizable=data.get("memoizable", False),
        )
