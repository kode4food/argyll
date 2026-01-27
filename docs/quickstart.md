# Quickstart

This quickstart uses a script step so you can run end-to-end without writing a separate step service.

## 1) Start the engine

```bash
docker compose up valkey argyll-engine
```

The engine will be available at http://localhost:8080.

## 2) Register a script step

```bash
curl -X POST http://localhost:8080/engine/step \
  -H "Content-Type: application/json" \
  -d '{
    "id": "hello-script",
    "name": "Hello Script",
    "type": "script",
    "attributes": {
      "name": {"role": "required", "type": "string"},
      "greeting": {"role": "output", "type": "string"}
    },
    "script": {"language": "ale", "script": "{:greeting name}"}
  }'
```

## 3) Start a flow

```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "hello-flow",
    "goals": ["hello-script"],
    "init": {"name": "Argyll"}
  }'
```

## 4) Inspect flow state

```bash
curl http://localhost:8080/engine/flow/hello-flow
```

Look for:

- `status`: should become `completed`
- `attributes`: should include `greeting: "Argyll"`

## 5) Next steps

- Build a real HTTP step: ./guides/step-types.md
- Async steps and webhooks: ./guides/async-steps.md
- Go SDK: ./go/README.md
