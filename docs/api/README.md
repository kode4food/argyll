# API Documentation

OpenAPI specifications for the Argyll HTTP APIs.

## Specifications

- **[engine-api.yaml](engine-api.yaml)** - Engine HTTP API for managing flows and steps
- **[step-interface.yaml](step-interface.yaml)** - HTTP contract for step implementations

## Viewing the Docs

### Swagger UI
```bash
npm install -g swagger-ui-watcher
swagger-ui-watcher engine-api.yaml
```

### Online
Upload to [Swagger Editor](https://editor.swagger.io/) or [Redoc](https://redocly.github.io/redoc/)

## Quick Start

If you are new, start with [docs/quickstart.md](../quickstart.md) and then return here for API reference.

### Engine API

Register a step and start a flow:

```bash
# Register step
curl -X POST http://localhost:8080/engine/step \
  -H "Content-Type: application/json" \
  -d '{
    "id": "text-processor",
    "name": "Text Processor",
    "type": "sync",
    "attributes": {
      "input_text": {"role": "required", "type": "string"},
      "output_text": {"role": "output", "type": "string"}
    },
    "http": {
      "method": "POST",
      "endpoint": "http://localhost:8081/process",
      "timeout": 30000
    }
  }'

# Start flow
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "wf-001",
    "goals": ["text-processor"],
    "init": {"input_text": ["Hello"]}
  }'

# Check status
curl http://localhost:8080/engine/flow/wf-001/status
```

### Step Interface

Steps receive requests like:

For all HTTP methods, the engine resolves endpoint placeholders from runtime inputs before dispatch.
For `POST`, `PUT`, and `DELETE`, the engine sends input arguments as the JSON body.
For `GET`, no JSON body is sent.

Execution metadata is sent in headers:

```http
Argyll-Flow-ID: wf-001
Argyll-Step-ID: text-processor
Argyll-Receipt-Token: token_abc
Argyll-Webhook-URL: http://localhost:8080/webhook/wf-001/text-processor/token_abc
```

```json
{
  "input_text": "Hello"
}
```

Successful steps return output arguments directly:

```json
{
  "output_text": "HELLO"
}
```

Failed steps return a non-2xx status and `application/problem+json`:

```json
{
  "type": "about:blank",
  "title": "Unprocessable Entity",
  "status": 422,
  "detail": "Error message"
}
```

## Async Webhooks

For async steps, the engine provides a webhook URL in `Argyll-Webhook-URL`. Post output arguments to that URL when work completes, or post Problem Details with `Content-Type: application/problem+json` when work fails. The token in the URL identifies the work item.
