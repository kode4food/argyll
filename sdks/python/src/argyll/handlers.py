"""Step execution context and server setup."""

import os
import time
import traceback
from dataclasses import dataclass
from typing import TYPE_CHECKING, Any, Callable

import requests
from flask import Flask, jsonify, request

from .errors import ClientError, HTTPError, StepRegistrationError, WebhookError
from .types import Args, Metadata, StepID, StepResult

if TYPE_CHECKING:
    from .builder import StepBuilder
    from .client import Client, FlowClient


MAX_REGISTRATION_ATTEMPTS = 5
BACKOFF_MULTIPLIER_SECONDS = 2


@dataclass(frozen=True)
class StepContext:
    """Context provided to step handlers."""

    client: "FlowClient"
    step_id: StepID
    metadata: Metadata


@dataclass(frozen=True)
class AsyncContext:
    """Context for async step execution with webhook support."""

    context: StepContext
    webhook_url: str

    @property
    def client(self) -> "FlowClient":
        """Get flow client."""
        return self.context.client

    @property
    def step_id(self) -> StepID:
        """Get step ID."""
        return self.context.step_id

    @property
    def metadata(self) -> Metadata:
        """Get metadata."""
        return self.context.metadata

    @property
    def flow_id(self) -> str:
        """Get flow ID."""
        return self.context.client.flow_id

    def success(self, outputs: Args) -> None:
        """Mark async step as successful."""
        result = StepResult(success=True, outputs=outputs)
        self._send_webhook(result)

    def fail(self, error: str) -> None:
        """Mark async step as failed."""
        result = StepResult(success=False, error=error)
        self._send_webhook(result)

    def complete(self, result: StepResult) -> None:
        """Complete async step with full result."""
        self._send_webhook(result)

    def _send_webhook(self, result: StepResult) -> None:
        """Send webhook to engine."""
        try:
            resp = requests.post(
                self.webhook_url,
                json=result.to_dict(),
                headers={"Content-Type": "application/json"},
                timeout=30,
            )
            resp.raise_for_status()
        except requests.RequestException as e:
            raise WebhookError(
                f"Failed to send webhook to {self.webhook_url}: {e}"
            ) from e


StepHandler = Callable[[StepContext, Args], StepResult]


def create_step_server(
    client: "Client", builder: "StepBuilder", handler: StepHandler
) -> None:
    """Create and start Flask server for step execution."""
    port_str = os.getenv("STEP_PORT", "8081")
    port = int(port_str)
    hostname = os.getenv("STEP_HOSTNAME", "localhost")

    step_id = builder._id
    endpoint = f"http://{hostname}:{port}/{step_id}"
    health_endpoint = f"http://{hostname}:{port}/health"

    builder = builder.with_endpoint(endpoint).with_health_check(health_endpoint)

    step = builder.build()
    registered = False
    for attempt in range(1, MAX_REGISTRATION_ATTEMPTS + 1):
        try:
            if builder._dirty:
                client.update_step(step)
            else:
                try:
                    client.register_step(step)
                except ClientError as e:
                    if e.status_code != 409:
                        raise
                    client.update_step(step)
            registered = True
            break
        except Exception:
            if attempt >= MAX_REGISTRATION_ATTEMPTS:
                raise
            time.sleep(attempt * BACKOFF_MULTIPLIER_SECONDS)

    if not registered:
        raise StepRegistrationError("Failed to register step after retries")

    app = Flask(__name__)

    @app.route("/health", methods=["GET"])
    def health() -> Any:
        return jsonify({"status": "healthy", "service": step_id})

    @app.route(f"/{step_id}", methods=["POST"])
    def handle_step() -> Any:
        try:
            req_data = request.get_json()
            if not req_data:
                return jsonify({"error": "Invalid JSON"}), 400

            arguments = req_data.get("arguments", {})
            metadata = req_data.get("metadata", {})

            flow_id = metadata.get("flow_id", "")
            flow_client = client.flow(flow_id)

            ctx = StepContext(
                client=flow_client, step_id=step_id, metadata=metadata
            )

            result = _execute_with_recovery(ctx, handler, arguments)

            return jsonify(result.to_dict())

        except HTTPError as e:
            return jsonify({"error": e.args[0]}), e.status_code
        except Exception as e:
            return jsonify({"error": str(e)}), 500

    print(f"Starting step server: {step_id}")
    print(f"  Endpoint: {endpoint}")
    print(f"  Health: {health_endpoint}")

    app.run(host="0.0.0.0", port=port)


def _execute_with_recovery(
    ctx: StepContext, handler: StepHandler, args: Args
) -> StepResult:
    """Execute handler with panic recovery."""
    try:
        return handler(ctx, args)
    except HTTPError:
        raise
    except Exception as e:
        tb = traceback.format_exc()
        print(f"Step handler error: {tb}")
        return StepResult(success=False, error=f"Step handler panicked: {e}")
