# Async Steps and Webhooks

Async steps let you return immediately and complete later via webhook. Use them for long-running work, queue-based systems, or any flow that should not block on a single HTTP request. This is a core pattern in a goal-driven orchestrator where planning and execution are decoupled.

## How it works

1) Engine calls your step endpoint with arguments and metadata.
2) Your handler returns HTTP 200 quickly.
3) Your async worker POSTs a StepResult to the webhook URL.

The webhook URL format is:

```
{WEBHOOK_BASE_URL}/webhook/{flow_id}/{step_id}/{token}
```

The engine provides this URL in `metadata.webhook_url` for async steps.

## Request and response shape

Your step endpoint receives the same inputs as a sync step. The initial response should be a standard StepResult:

```json
{
  "success": true,
  "outputs": {}
}
```

The webhook uses the same StepResult payload shape:

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

Each async work item has a token. The webhook URL includes that token, and the engine uses it to update the correct work item. Never reuse a token for multiple completions.

## Idempotency

Async completion should be idempotent. If your worker retries a webhook, the engine should receive the same payload. Avoid emitting multiple completions for different outcomes using the same token.

## Retries and failures

- Retry behavior is controlled by step work config.
- A webhook failure should return `success: false` with an error string.
- If a work item reports not completed, the engine can schedule a retry using backoff.

See [guides/retries.md](./retries.md) for details.

## Operational checklist

- Ensure `WEBHOOK_BASE_URL` is reachable from your workers.
- Log the flow ID, step ID, and token in your worker for traceability.
- Make webhook handlers resilient and fast to avoid timeouts and duplicate sends.
- Treat webhook calls as at-least-once delivery.

## Example webhook call

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"success":true,"outputs":{"result":"ok"}}' \
  https://your-webhook-base/webhook/wf-123/step-abc/token-xyz
```
