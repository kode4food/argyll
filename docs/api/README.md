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
    "init": {"input_text": "Hello"}
  }'

# Check status
curl http://localhost:8080/engine/flow/wf-001
```

### Step Interface

Steps receive requests like:

```json
{
  "arguments": {
    "input_text": "Hello"
  },
  "metadata": {
    "flow_id": "wf-001",
    "step_id": "text-processor",
    "receipt_token": "token_abc"
  }
}
```

And return:

```json
{
  "success": true,
  "outputs": {
    "output_text": "HELLO"
  }
}
```

Or errors:

```json
{
  "success": false,
  "error": "Error message"
}
```
