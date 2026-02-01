# Retries and Backoff

Retries are per-work-item and configured per step. They apply when a work item reports not completed or fails with a retryable error. In Argyll, retries are part of the orchestrator’s control loop, not application code.

## When retries are scheduled

Retries are scheduled when a work item reports not completed to the engine. This happens in two ways:

1. **Explicit `work_not_completed`**: Your step handler returns `success: false` with an error in the response payload. If you want the engine to retry, you must indicate this is a transient failure (not a permanent error).

2. **Async webhook with `success: false`**: Your async worker completes via webhook with `success: false` and an error message.

If a step returns `success: true`, the work item is considered complete regardless of whether actual work succeeded. This is a design choice: the engine respects your step handler's judgment about whether something should be retried.

**Important:** The engine does NOT examine HTTP status codes, network errors, or timeouts. Your step handler is responsible for deciding what constitutes a retryable error and returning `success: false` accordingly.

## Configuration

Retry configuration lives in the step's `work_config`:

- `max_retries`
- `backoff` (milliseconds)
- `max_backoff` (milliseconds)
- `backoff_type` (fixed, linear, exponential)

Semantics:

- `max_retries = 0` disables retries
- `max_retries > 0` limits the number of attempts
- `max_retries = -1` allows unlimited retries

## Backoff strategies

- Fixed: constant delay between attempts
- Linear: delay grows by a fixed increment each attempt
- Exponential: delay doubles each attempt up to `max_backoff` (milliseconds)

Backoff is applied per work item, not per step.

## Work item lifecycle with retries

1) Work item starts
2) Work item fails or is not completed
3) Engine schedules retry with next retry time
4) Retry queue triggers a new work start when the time arrives

The step completes only when all work items succeed or a failure becomes permanent.

## Terminal failures

If a work item fails permanently, the step fails. If a goal step fails, the flow fails. Non-goal step failure can still cause other steps to become unreachable, which may fail the flow depending on the execution plan.

## Design tips

- Use small fixed backoff for quick retry of flaky dependencies.
- Use exponential backoff when dealing with rate limits or unstable services.
- Keep `max_retries` low unless your step is idempotent and you can tolerate long recovery times.
- Prefer reporting `work_not_completed` for work that is genuinely in progress.

## Observability

Retry scheduling is recorded as events, so you can reconstruct when and why retries happened by replaying the event log. This makes retries explainable rather than opaque.
