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

## Idempotency

Each work item has a unique **receipt_token** in the metadata. The engine uses this token to ensure idempotency: duplicate webhook calls with the same token are rejected, making it safe to retry.

This means **your worker can safely retry the webhook call**—duplicate attempts are automatically rejected without creating duplicate work.

**Example:**
```bash
# First attempt (succeeds but response is lost)
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"success":true,"outputs":{"result":"ok"}}' \
  https://engine/webhook/wf-123/step-abc/receipt-token-xyz
# HTTP 200 - work item marked as succeeded

# Worker retries the same request (same token)
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"success":true,"outputs":{"result":"ok"}}' \
  https://engine/webhook/wf-123/step-abc/receipt-token-xyz
# HTTP 400 Bad Request - already processed
# Your worker can safely give up and move on
```

**What you SHOULD do:**
- Store the receipt_token for logging and debugging
- Implement retry logic with exponential backoff in your worker
- Treat 400 responses as terminal and inspect the error body
- Treat 5xx responses and network errors as retryable

**What you DON'T need to do:**
- Implement your own deduplication—the engine handles it
- Track which tokens were processed—the engine rejects duplicates
- Return the same result on retry—the engine enforces it

**Safe webhook call pattern:**
```python
def send_completion(flow_id, step_id, receipt_token, result):
    url = f"https://engine/webhook/{flow_id}/{step_id}/{receipt_token}"

    # Simple retry loop with backoff
    backoff = 1
    for attempt in range(5):
        try:
            response = requests.post(url, json=result, timeout=10)

            if response.status_code == 200:
                return True  # Success
            elif response.status_code == 400:
                # Terminal request error (invalid flow/step/token/work state)
                # Do not retry blindly; inspect response and alert/log.
                log.error(f"Webhook rejected for {receipt_token}: "
                          f"{response.text}")
                return False
            else:
                # Retryable error (5xx, network, timeout)
                raise Exception(f"HTTP {response.status_code}")

        except Exception as e:
            if attempt < 4:
                time.sleep(backoff)
                backoff *= 2
            else:
                raise
```

## Retries and failures

- Retry behavior is controlled by step work config.
- Posting webhook payload `{ "success": false, "error": "..." }` records a
  permanent failure for that work item.
- Retry scheduling is driven by `work_not_completed` transitions (for example,
  transient invocation failures like network errors or HTTP 5xx from a step
  endpoint).

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
