# Dev Setup

This guide covers the local development loop for the goal-driven orchestrator, UI, and step services.

## Docker Compose (recommended)

```bash
docker compose up
```

This starts:

- Argyll engine on http://localhost:8080
- UI on http://localhost:3001
- Chat service on http://localhost:8072
- Keycloak on http://localhost:8180, with demo users `reader`, `operator`, and `admin` whose passwords match their names
- Anthropic, Gemini, Amazon Bedrock, and OpenAI Agent workers on the Compose network
- Example step services (ports 8081–8086)

Docker Compose enables Keycloak authentication by default. Sign in through Keycloak to use the UI. Run `AUTH_PROVIDER=none docker compose up` to disable authentication for local development.

The examples use the `demo-services` service account. The engine uses `argyll-engine` to dispatch work, and the Agent workers use `agent-workers` to call MCP. The MCP server forwards the caller's token to the engine.

Before upgrading a deployment that issued callback URLs without secrets, stop starting new Flows and let outstanding async work settle. Previously issued URLs lack the secret required by the upgraded callback route.

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
- `CALLBACK_MAX_BYTES` (largest Step callback body, 10 MiB by default)
- `ARGYLL_TOKEN_URL`, `ARGYLL_CLIENT_ID`, `ARGYLL_CLIENT_SECRET` (the engine's service account for agent workers)
- `AGENT_WORKERS` (agent worker for each provider, as comma-separated `provider=url` pairs, with `*` serving every provider not listed)
- `RAFT_NODE_ID`, `RAFT_ADDRESS`, and `RAFT_DATA_DIR` (single-node or cluster identity)
- `RAFT_LOG_TAIL_SIZE` (number of recent write-ahead log entries retained in memory, default `20480`)
- `RAFT_SERVERS` (initial voting members of a multi-node cluster)

If you run the engine outside Docker, make sure these are set appropriately for your host and network.

## Running the reference Agent workers

Docker Compose starts `argyll-agent-anthropic`, `argyll-agent-gemini`, `argyll-agent-bedrock`, and `argyll-agent-openai` automatically. Each offers the `argyll` MCP server for you to select in a Step. To enable a provider, add its credentials to that worker's file at the repository root:

- Anthropic uses `ANTHROPIC_API_KEY` in `.env.d/agent-anthropic.env`.
- Gemini uses `GEMINI_API_KEY` in `.env.d/agent-gemini.env`.
- Amazon Bedrock uses `AWS_REGION` and AWS credentials in `.env.d/agent-bedrock.env`.
- OpenAI uses `OPENAI_API_KEY` in `.env.d/agent-openai.env`.

The Anthropic worker also accepts `ANTHROPIC_AUTH_TOKEN` in place of `ANTHROPIC_API_KEY` for gateways that expect a bearer token. The API key wins when both are set.

The Gemini worker offers the Gemini API's text models as the `gemini` provider through the Interactions API. It lists every model that generates content and leaves out deprecated models and models that produce audio, images, video, or music. Each attempt keeps its conversation on the worker, so Google stores no interaction history. Effort settings map to Gemini thinking levels.

The Bedrock worker offers the region's text models that support tool calling as the `bedrock` provider. It lists on-demand models and active inference profiles, leaving out retired models. Models that accept image input are included. Models that produce images, audio, video, speech, or embeddings are left out. Its role needs `bedrock:InvokeModel`, `bedrock:ListFoundationModels`, and `bedrock:ListInferenceProfiles`. It reads credentials from the standard AWS sources: environment variables such as `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`, shared config files with `AWS_PROFILE`, or an attached role. Set `AWS_BEARER_TOKEN_BEDROCK` to use a Bedrock API key instead. Set `AWS_ENDPOINT_URL_BEDROCK_RUNTIME` and `AWS_ENDPOINT_URL_BEDROCK` to reach Bedrock through VPC endpoints.

Set custom `PROVIDER_URLS` in the file for the worker serving those endpoints. Use the optional `.env.d/agent.env` for settings all workers share. Worker-specific values take precedence. The `.env.d/` directory is excluded from Git and Docker builds.

Run `docker compose up -d argyll-agent-anthropic argyll-agent-gemini argyll-agent-bedrock argyll-agent-openai` to apply the new environment.

The workers share the Go module in `workers/agent`. From the repository root, run `go run ./workers/agent/cmd/argyll-agent-anthropic`, `go run ./workers/agent/cmd/argyll-agent-gemini`, `go run ./workers/agent/cmd/argyll-agent-bedrock`, or `go run ./workers/agent/cmd/argyll-agent-openai`. To run several locally, set a different `PORT` for each process and configure `AGENT_WORKERS` with their URLs. To build or test with the Go workspace disabled, run `GOWORK=off go build -mod=readonly ./cmd/...` or `make test` from `workers/agent`. The module uses the engine contract from the same checkout.

Run `make -C workers/agent refresh-anthropic-models`, `refresh-gemini-models`, `refresh-bedrock-models`, or `refresh-openai-models` to refresh that worker's bundled JSON from the provider's published information. Each command writes `model-prices.json`. Anthropic also writes `prefix-thresholds.json`, and Gemini, Bedrock, and OpenAI also write `model-capabilities.json`. Gemini prices change on published dates, and its refresh records the rates in effect on the day it runs. The bundled Bedrock prices are for `us-east-1`.

The refresh commands are the only part of the workers that read public documentation. At runtime, workers use only provider APIs and local files. A refresh validates what it reads and replaces files only when every output is complete. The Gemini refresh records each model's thinking levels and leaves out media and deprecated models. The Bedrock refresh records supported and excluded models, effort levels, and long-context price tiers. It also reports the models it could not price. On Bedrock, Claude, Nova, and OpenAI models accept effort settings. An unsupported effort value fails the Step permanently.

Gemini, Bedrock, and OpenAI list every text model their provider offers, leaving out the models their capability tables hide. New text models appear without a refresh. To change a model's listing, add it to `MODEL_CAPABILITIES` with a `provider/model` key. Set `hidden` to `true` to leave it out, or list its `efforts`. For the OpenAI worker, set `endpoint` to `responses` to use the Responses API instead of Chat Completions. Workers ask the model for text and tool calls, and prompt it until it returns one final result.

Workers discover capabilities when they start. Bedrock keeps its last successful list of inference profiles, so profile IDs and ARNs keep resolving after a failed discovery. A request that arrives before any profile list is available fails and is retried.

If Bedrock is missing from the Step editor's provider list, check that `.env.d/agent-bedrock.env` exists and sets `AWS_REGION`. The worker lists Bedrock once a region is set. Supply AWS credentials through that file or another standard AWS credential source inside the container, then recreate the worker with `docker compose up -d argyll-agent-bedrock`. Listing models also requires the permissions above.

To use a host profile, set `AWS_PROFILE` in `.env.d/agent-bedrock.env` and mount its credentials file read-only in a local `docker-compose.override.yml`:

```yaml
services:
  argyll-agent-bedrock:
    volumes:
      - "${HOME}/.aws/credentials:/root/.aws/credentials:ro"
```

For an `aws login` profile, mount the shared config read-only and the login cache writable instead of the credentials file. The SDK refreshes the session through that cache. Keep these mounts in the ignored local override:

```yaml
services:
  argyll-agent-bedrock:
    volumes:
      - "${HOME}/.aws/config:/root/.aws/config:ro"
      - "${HOME}/.aws/login/cache:/root/.aws/login/cache"
```

If Bedrock appears with no models, inspect the discovery error in the Step editor or `docker compose logs argyll-agent-bedrock`. `UnrecognizedClientException` means AWS rejected the credentials. Refresh the profile and recreate the worker. `AccessDeniedException` requires checking the role's model-listing permissions.

To produce copies outside the checkout, run the refresh command directly with `-dir`, then point `MODEL_PRICES`, `MODEL_CAPABILITIES`, and `PREFIX_THRESHOLDS` at the files it writes. For example, from `workers/agent`, run `GOWORK=off go run ./cmd/argyll-agent-bedrock/internal/refresh -dir /path/to/defaults -region eu-north-1`.

Each Agent worker includes rates for its provider. To replace a model's rates or add a model, set `MODEL_PRICES` to a JSON file readable by the worker. Use `provider/model` keys and rates per million tokens for `input`, `output`, `cache_input`, and `cache_output`, with zero for free tokens. Optional `tiers` entries set an `input_over` cutoff and the rates for turns above it. A model override replaces its base rates and tiers. Provider-reported charges take precedence. See the [Agent Worker Contract](api/agent-worker.yaml) for the usage fields.

Automated tests, including `make test` and GitHub Actions, use local provider and MCP fixtures. They must not require live provider access or credentials. The `argyll-agent-check` command is a separate, explicitly invoked live check and must not be added to automated test targets.

Capability responses include discovery `problems` alongside available providers. The Step editor shows these failures with their source and status instead of presenting an empty picker without an explanation. Worker logs retain the underlying error as well.

## UI dev loop

```bash
cd web
npm install
npm run dev
```

The UI uses `VITE_API_URL` for the engine (default `http://localhost:8080`) and `VITE_CHAT_URL` for the standalone chat service (default `http://localhost:8072`). The chat service forwards requests to Agent workers. See the [Web UI chat contract](api/chat.yaml) for the interface between the UI and chat app.

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
