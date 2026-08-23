# Compensation

Compensation is the mechanism Argyll uses to undo previously completed work when a step ultimately fails. If a step processes multiple work items and some succeed before others fail, the engine calls a configured `compensate` endpoint for every work item that succeeded.

Compensation is available for sync and async HTTP steps. Script and flow steps cannot be compensated.

## When compensation fires

By default compensation is triggered at **step failure**, not flow failure. When a step fails (permanently, after retries are exhausted for all failing work items), the engine raises a `step_failed` event and immediately schedules compensation for every work item whose status is `succeeded`. Steps that already completed successfully are left alone; see [Rollback on Failure](#rollback-on-failure) to roll those back too.

Compensation continues even after the flow has reached terminal state (`failed`). A caller may act on the failure immediately while compensation is still producing side effects (changes to external systems). The flow is not deactivated until all compensation work items finish.

## Rollback on Failure

Step-level compensation undoes only the failing step's own work. A flow that fails at its fifth step leaves the side effects of steps one through four in place. Enable **Rollback on Failure** in Start Flow or on a Flow Step to get saga-style rollback instead. Both controls set the API's `compensate` field: on a root flow request or under `flow` for a child flow. Once the flow reaches `failed`, every succeeded work item in the flow is compensated, not just those of the step that failed.

```json
{
  "id": "wf-checkout-123",
  "goals": ["ship-order"],
  "compensate": true,
  "init": {
    "order_id": ["ord-abc"]
  }
}
```

Only steps with `handling: "compensated"` participate. Enabling **Rollback on Failure** on a flow with no compensated steps does nothing and is not reported as an error.

Work items that complete *after* the flow has failed are still compensated. This matters when a slow step is still in flight at the moment another step fails: its side effect is recorded, then undone when its step's wave comes up. The flow is not deactivated until all of it settles.

Steps are unwound in reverse dependency order, running the execution plan's own parallelism backwards. The engine releases one wave at a time: a step compensates once nothing that consumed its outputs is still waiting to compensate. Steps that could have run in parallel unwind in parallel, and the next wave does not start until every compensation in the current one reaches a terminal state.

Arrows show execution order. Unwinding runs in reverse, one wave at a time:

```
reserve-inventory ─┐
                   ├─→ charge-card ─→ ship-order
validate-address  ─┘

unwind:  ship-order          (wave 1)
         charge-card         (wave 2)
         reserve-inventory,
         validate-address    (wave 3, together)
```

Work items within a single step are independent and compensate in parallel with each other.

A step without compensated handling is skipped rather than treated as a wave of its own, so it never delays the steps behind it.

Ordering applies to flow-level rollback only. A failing step's cleanup of its own succeeded work items starts as soon as the step fails, without waiting for a wave, because nothing downstream of it can have run.

### Child flows are sealed

A flow step's child flow does **not** inherit the parent's **Rollback on Failure** setting. Whether a child flow rolls back is decided by its own Flow Step definition:

```json
{
  "id": "reserve-and-charge",
  "name": "Reserve and Charge",
  "type": "flow",
  "flow": {
    "goals": ["charge-card"],
    "compensate": true
  },
  "attributes": {
    "order_id": { "role": "required", "type": "string" }
  }
}
```

A parent that fails will not roll back a child flow that completed successfully, and a child flow that fails rolls back according to its own **Rollback on Failure** (`flow.compensate`) setting only.

A child flow reports its result to its parent as soon as its own work finishes, but is not deactivated until the parent releases it. A child that shows as terminal-but-active is waiting on its parent, not stuck.

## Configuring a compensate endpoint

Set the step's handling to `compensated`, configure the endpoint, and mark only the attributes it needs:

```json
{
  "id": "charge-card",
  "name": "Charge Card",
  "type": "async",
  "handling": "compensated",
  "http": {
    "invoke": {
      "endpoint": "https://payment-service.example.com/charge",
      "timeout": 1000
    },
    "compensate": {
      "endpoint": "https://payment-service.example.com/refund/{charge_id}"
    }
  },
  "attributes": {
    "amount": { "role": "required", "type": "number" },
    "charge_id": {
      "role": "output",
      "type": "string",
      "compensated": true
    }
  }
}
```

`compensated: true` selects an attribute for compensation. Unmarked attributes are not sent. The request body and `{param}` placeholders use each selected attribute's invocation-mapped inner name. If two selected attributes have the same inner name, choose one; selecting both is invalid.

`compensate` accepts its own `method` and `timeout`. The method defaults to `POST`; set `"method": "DELETE"` when the service models the undo as a resource deletion.

Timeouts resolve in three steps: the compensate `timeout` if set, otherwise the `invoke` timeout, otherwise the global engine timeout (`STEP_TIMEOUT`). Set a compensate timeout only when undoing takes materially longer or shorter than doing. Leaving it unset keeps the step's single timeout governing both.

## What the engine sends

The engine sends only attributes marked `compensated` as one flattened object using their invocation-mapped names:

```json
{
  "charge_id": "ch_abc123"
}
```

`GET` and `DELETE` compensation requests carry no body; selected attributes still resolve URL placeholders. `POST` and `PUT` carry the flattened payload.

The same `Argyll-Flow-ID`, `Argyll-Step-ID`, and `Argyll-Receipt-Token` headers sent to the work endpoint are also sent to the compensate endpoint. Use the receipt token as the idempotency key for compensation side effects.

## Retry behavior

Compensation uses the same `work_config` retry settings as the step's normal work execution. The engine treats compensation `5xx` responses (and transport errors) as temporary failures and schedules a `comp_retry_scheduled` event using the configured retry delay strategy. `4xx` responses are treated as permanent compensation failures.

When `max_retries` is exhausted, the compensation is marked `compensation_failed`.

```json
{
  "id": "reserve-inventory",
  "name": "Reserve Inventory",
  "type": "sync",
  "handling": "compensated",
  "http": {
    "invoke": {
      "endpoint": "https://inventory.example.com/reserve",
      "timeout": 3000
    },
    "compensate": {
      "endpoint": "https://inventory.example.com/release/{reservation_id}",
      "method": "DELETE"
    }
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
    "reservation_id": {
      "role": "output",
      "type": "string",
      "compensated": true
    }
  }
}
```

## Handling is explicit

`standard`, `memoized`, and `compensated` are mutually exclusive handling modes. A compensation endpoint is required only for `compensated` handling and is rejected for the other modes. Likewise, `compensated: true` is valid on an attribute only when the step handling is `compensated`.

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

If the engine restarts while compensations are in flight, they are recovered automatically:

- Work items already in `compensating` state are rescheduled using their stored `NextRetryAt`.
- Work items still in `succeeded` state on a failed step (e.g., the engine crashed before compensation could start) are detected and compensation is started from the beginning.

## Design tips

- Compensation is not a substitute for idempotency. Implement idempotent compensate endpoints by keying on the receipt token.
- Use `max_retries: -1` with care for compensation: unlimited retries on a permanently unavailable service will block flow deactivation indefinitely, and hold back the unwind of every step that ran before it.
- If compensation is not meaningful for a step (no side effects), omit the `compensate` field rather than implementing a no-op endpoint.
- For multi-step flows where partial success is common, consider sequencing steps so that compensatable steps run last.

## Related

- [Retries and Backoff](./retries.md)
- [Work Items](./work-items.md)
- [Async Steps](./async-steps.md)
