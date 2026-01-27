# Async Steps and Webhooks

Async steps let you return immediately and complete later via webhook.

## How it works

1) Engine calls your step endpoint with arguments and metadata.
2) Your handler returns HTTP 200 quickly.
3) Your async worker posts a StepResult to the webhook URL.

The webhook URL format is:

```
{WEBHOOK_BASE_URL}/webhook/{flow_id}/{step_id}/{token}
```

The engine provides this URL in `metadata.webhook_url` for async steps.

## StepResult payload

Send the same payload shape as a sync response:

```json
{
  "success": true,
  "outputs": {"key": "value"}
}
```

On failure:

```json
{
  "success": false,
  "error": "message"
}
```

## Token semantics

Each async work item has a token. The webhook URL includes that token, and the engine uses it to update the correct work item.

## Common pitfalls

- `WEBHOOK_BASE_URL` must be reachable from your step runtime.
- Do not reuse a token for multiple completions.
- Return JSON that matches `api.StepResult`.
