# Retries and Backoff

Retries apply to work items that return `work_not_completed` or encounter retryable errors.

## When retries are scheduled

Retries are scheduled when a work item reports not completed. The engine records a retry schedule and pushes the item into the retry queue.

## Configuration

Retry configuration lives in the step's `work_config`:

- `max_retries`
- `backoff_ms`
- `max_backoff_ms`
- `backoff_type` (fixed, linear, exponential)

## Terminal failures

If a work item fails permanently, the step fails. If a goal step fails, the flow fails.

## Tips

- Use small fixed backoff for fast retries and exponential for flaky dependencies.
- Keep `max_retries` low for idempotent steps unless you can tolerate long recovery times.
