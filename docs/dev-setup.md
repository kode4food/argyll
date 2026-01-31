# Dev Setup

This guide covers the local development loop for the goal-driven orchestrator, UI, and step services.

## Docker Compose (recommended)

```bash
docker compose up
```

This starts:

- Valkey (flow and engine stores)
- Argyll engine on http://localhost:8080
- UI on http://localhost:3001
- Example step services (ports 8081–8086)

Use this when you want a working system quickly without wiring services by hand.

## Running the engine locally (without Docker)

To run the engine binary directly (useful when iterating on engine code):

```bash
cd engine
make test
go run ./cmd/argyll
```

Ensure your environment points at a Valkey instance (local or Docker).

## Engine environment variables

Local defaults are defined in `docker-compose.yml`. The most important settings are:

- `ENGINE_REDIS_ADDR` and `FLOW_REDIS_ADDR` (Valkey endpoints)
- `API_HOST` and `API_PORT` (engine HTTP server)
- `WEBHOOK_BASE_URL` (used for async step callbacks)

If you run the engine outside Docker, make sure these are set appropriately for your host and network.

## UI dev loop

```bash
cd web
npm install
npm run dev
```

The UI expects `VITE_API_URL` to point at the engine (default: http://localhost:8080).

## Step service dev loop

You can implement steps as HTTP services in any language. For quick iteration:

1) Run the engine (Docker or local binary)
2) Run a step service (Go builder or your own HTTP server)
3) Register the step via API or builder
4) Start a flow and inspect state

## Validation checklist

- Engine reachable at http://localhost:8080
- UI reachable at http://localhost:3001
- Step services reachable from the engine
- `WEBHOOK_BASE_URL` reachable from your step runtime

## Troubleshooting

- If async callbacks never complete, verify `WEBHOOK_BASE_URL` is reachable from your step runtime and includes the correct host/port.
- If steps appear stuck in `active`, verify the step server can reach the engine and is returning valid JSON responses.
- If the UI is empty, verify the engine is running and `VITE_API_URL` matches your engine address.
