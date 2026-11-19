# Dev Setup

This guide covers the local development loop for the goal-driven orchestrator, UI, and step services.

## Docker Compose (recommended)

```bash
docker compose up
```

This starts:

- Argyll engine on http://localhost:8080
- UI on http://localhost:3001
- Anthropic and OpenAI Agent workers on the Compose network
- Example step services (ports 8081–8086)

Docker Compose configures the services and their connections for local development.

## Running the engine binary

To run the engine binary directly (useful when iterating on engine code):

```bash
cd engine
make test
go run ./cmd/argyll
```

This starts a single-node Raft-backed engine with the default local config.

To start the local 3-node dev cluster:

```bash
cd engine
./start.sh
```

## Engine environment variables

Local defaults are defined in `docker-compose.yml`. The most important settings are:

- `API_HOST` and `API_PORT` (engine HTTP server)
- `WEBHOOK_BASE_URL` (used for async step callbacks)
- `AGENT_WORKERS` (agent worker for each provider, as comma-separated `provider=url` pairs, with `*` serving every provider not listed)
- `RAFT_NODE_ID`, `RAFT_ADDRESS`, and `RAFT_DATA_DIR` (single-node or cluster identity)
- `RAFT_LOG_TAIL_SIZE` (number of recent write-ahead log entries retained in memory, default `20480`)
- `RAFT_SERVERS` (initial voting members of a multi-node cluster)

If you run the engine outside Docker, make sure these are set appropriately for your host and network.

## Running the reference Agent workers

Docker Compose starts `argyll-agent-anthropic` and `argyll-agent-openai` automatically. Both offer the `argyll` MCP server for you to select in a Step. To enable a provider, add its API key to that worker's file at the repository root:

- Anthropic uses `ANTHROPIC_API_KEY` in `.env.d/agent-anthropic.env`.
- OpenAI uses `OPENAI_API_KEY` in `.env.d/agent-openai.env`.

Set custom `PROVIDER_URLS` in the file for the worker serving those endpoints. Use the optional `.env.d/agent.env` for settings both workers share. Worker-specific values take precedence. The `.env.d/` directory is excluded from Git and Docker builds.

Run `docker compose up -d argyll-agent-anthropic argyll-agent-openai` to apply the new environment.

The workers share the Go module in `workers/agent`. From the repository root, run `go run ./workers/agent/cmd/argyll-agent-anthropic` or `go run ./workers/agent/cmd/argyll-agent-openai`. To run both locally, set a different `PORT` for each process and configure `AGENT_WORKERS` with their URLs. To build or test with the Go workspace disabled, run `GOWORK=off go build -mod=readonly ./cmd/...` or `make test` from `workers/agent`. The module uses the engine contract from the same checkout.

Run `make -C workers/agent refresh-anthropic-models` to refresh Anthropic prices and cache thresholds. Run `make -C workers/agent refresh-openai-models` to refresh OpenAI prices and model capabilities. Each command reads the provider's published documentation and updates that worker's bundled JSON.

Each Agent worker includes rates for its default provider. To replace a model's rates or add a model, set `MODEL_PRICES` to a JSON file readable by the worker. Use `provider/model` keys and rates per million tokens for `input`, `output`, `cache_input`, and `cache_output`, with zero for free tokens. Optional `tiers` entries set an `input_over` cutoff and the rates for turns above it. A model override replaces its base rates and tiers. Provider-reported charges take precedence. See the [Agent Worker Contract](api/agent-worker.yaml) for the usage fields.

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
