# Compensation

Compensation is the mechanism Argyll uses to undo previously completed work when
a step ultimately fails. If a step processes multiple work items and some succeed
before others fail, the engine calls a configured `compensate` endpoint for every
work item that succeeded.

Compensation is available for sync and async HTTP steps. Script and flow steps
cannot be compensated.

## When compensation fires

Compensation is triggered at **step failure**, not flow failure. When a step
fails (permanently, after retries are exhausted for all failing work items), the
engine raises a `step_failed` event and immediately schedules compensation for
every work item whose status is `succeeded`.

Compensation continues even after the flow has reached terminal state (`failed`). A caller may act on the failure immediately while compensation is still producing side effects (changes to external systems). The flow is not deactivated until all compensation work items finish.

## Configuring a compensate endpoint

Add `compensate` to the step's `http` configuration:

```json
{
  "id": "charge-card",
  "name": "Charge Card",
  "type": "async",
  "http": {
    "endpoint": "https://payment-service.example.com/charge",
    "timeout": 1000,
    "compensate": "https://payment-service.example.com/refund/{charge_id}"
  },
  "attributes": {
    "amount": { "role": "required", "type": "number" },
    "charge_id": { "role": "output", "type": "string" }
  }
}
```

The `compensate` URL supports `{param}` placeholders. Placeholders are resolved
from a merged view of the work item's inputs and outputs, with **outputs taking
priority** over inputs when names collide. This lets you reference output values
like `{charge_id}` directly in the URL.

## What the engine sends

The engine sends a `POST` request to the resolved compensate URL with:

```json
{
  "input": { "amount": 49.99 },
  "output": { "charge_id": "ch_abc123" }
}
```

The same `Argyll-Flow-ID`, `Argyll-Step-ID`, and `Argyll-Receipt-Token` headers
sent to the work endpoint are also sent to the compensate endpoint. Use the
receipt token as the idempotency key for compensation side effects.

## Retry behavior

Compensation uses the same `work_config` retry settings as the step's normal work execution. The engine treats compensation `5xx` responses (and transport errors) as temporary failures and schedules a `comp_retry_scheduled` event using the configured retry delay strategy. `4xx` responses are treated as permanent compensation failures.

When `max_retries` is exhausted, the compensation is marked `compensation_failed`.

```json
{
  "id": "reserve-inventory",
  "name": "Reserve Inventory",
  "type": "sync",
  "http": {
    "endpoint": "https://inventory.example.com/reserve",
    "timeout": 3000,
    "compensate": "https://inventory.example.com/release/{reservation_id}"
  },
  "work_config": {
    "max_retries": 5,
    "init_backoff": 500,
    "max_backoff": 30000,
    "backoff_type": "exponential"
  },
  "attributes": {
    "sku": { "role": "required", "type": "string" },
    "quantity": { "role": "required", "type": "number" },
    "reservation_id": { "role": "output", "type": "string" }
  }
}
```

## Memoizable steps cannot be compensated

Steps with `memoizable: true` assume their work has no observable side effects,
so compensation is not allowed. Configuring both `memoizable: true` and a
`compensate` URL is a validation error.

## Work item states

Compensation adds three work item states to the normal lifecycle:

| Status | Meaning |
|--------|---------|
| `compensating` | Compensation dispatched, waiting for result |
| `compensated` | Compensation completed successfully |
| `compensation_failed` | Compensation permanently failed |

These are work-item status values. Compensation events use the shorter `comp_*` names: `comp_started`, `comp_succeeded`, `comp_retry_scheduled`, and `comp_failed`. In particular, a `comp_failed` event sets the work-item status to `compensation_failed`.

The flow is not deactivated until all compensation work items reach a terminal state (`compensated` or `compensation_failed`).

## Startup recovery

If the engine restarts while compensations are in flight, they are recovered
automatically:

- Work items already in `compensating` state are rescheduled using their stored
  `NextRetryAt`.
- Work items still in `succeeded` state on a failed step (e.g., the engine
  crashed before compensation could start) are detected and compensation is
  started from the beginning.

## Design tips

- Compensation is not a substitute for idempotency. Implement idempotent
  compensate endpoints by keying on the receipt token.
- Use `max_retries: -1` with care for compensation: unlimited retries on a
  permanently unavailable service will block flow deactivation indefinitely.
- If compensation is not meaningful for a step (no side effects), omit the
  `compensate` field rather than implementing a no-op endpoint.
- For multi-step flows where partial success is common, consider sequencing
  steps so that compensatable steps run last.

## Related

- [Retries and Backoff](./retries.md)
- [Work Items](./work-items.md)
- [Async Steps](./async-steps.md)
