# Dev Setup

This guide covers the local development loop for the engine and step services.

## Docker Compose (recommended)

```bash
docker compose up
```

This starts:

- Valkey (flow and engine stores)
- Argyll engine on http://localhost:8080
- UI on http://localhost:3001
- Example step services (ports 8081–8086)

## Engine environment variables

Local defaults are defined in `docker-compose.yml`. The most important settings are:

- `ENGINE_REDIS_ADDR` and `FLOW_REDIS_ADDR` (Valkey endpoints)
- `API_HOST` and `API_PORT` (engine HTTP server)
- `WEBHOOK_BASE_URL` (used for async step callbacks)

If you run the engine outside Docker, make sure these are set appropriately.

## Local dev loop

1) Run the engine (docker compose or binary)
2) Run a step service (Go builder or your own HTTP server)
3) Register the step via API or builder
4) Start a flow and inspect state

## Troubleshooting

- If async callbacks never complete, verify `WEBHOOK_BASE_URL` is reachable from your step runtime and includes the correct host/port.
- If steps appear stuck in `active`, verify the step server can reach the engine and is returning valid JSON responses.
- If the UI is empty, verify the engine is running and `VITE_API_URL` matches your engine address.
