# Argyll

Distributed, goal-oriented flow processing system using lazy evaluation and event sourcing. Steps declare input/output dependencies; the engine orchestrates execution based on state availability, executing only the minimal set of steps needed to reach specified goals.

## Build & Test

```bash
# Go
cd engine && make test

# TypeScript
cd web && npm run format && npm test && npm run lint && npm run type-check
```

## Docker

```bash
docker compose up                      # All services
docker compose up valkey argyll-engine # Core only
docker compose logs -f argyll-engine   # Follow logs
```

## Package Structure

```
engine/
  internal/
    engine/      # Core orchestration (engine.go, flow.go, flow-exec.go, step.go, work-exec.go)
    events/      # Event sourcing types and projections
    client/      # HTTP client for step invocation
    config/      # Configuration management
    server/      # HTTP/WebSocket API (Gin-based)
  pkg/
    api/         # Public types and interfaces
    builder/     # Step builder utilities
  cmd/argyll/    # Main entry point
  tests/         # Integration tests
web/             # React UI (atomic design)
examples/        # Sample step implementations
```

## Implementation Status

### Fully Implemented

- Distributed coordination (optimistic concurrency)
- Event sourcing (Valkey backend)
- Lazy evaluation (goal-oriented execution)
- Step types (sync HTTP, async HTTP, script)
- Immutable execution plans
- Real-time UI (React 19 + WebSocket)
- Health monitoring
- Separate engine/flow stores
- Step retry with configurable backoff
- Flow archiving (Redis stream consumption)

### Partial

- Script security: sandboxed but no resource limits
- Input validation: UI correct, server permissive

### Not Implemented

- Flow pending state (immediate activation)
- Metrics/observability (no Prometheus/tracing)

## API

- Engine API: `/engine/docs/engine-api.yaml`
- Step Interface: `/engine/docs/step-interface.yaml`
- Base path: `/engine/`

### Step Request/Response

```json
// Request
{
  "step_id": "unique-id",
  "arguments": { "key": "value" },
  "meta": { "flow_id": "wf-123" }
}

// Response
{
  "success": true,
  "outputs": { "result": "value" }
}
```

## Web UI

- Location: `/web`
- Tech: React 19 + TypeScript + Tailwind CSS v3
- Port: 3001
- API: Connects to engine at `http://localhost:8080`

Features: Dashboard, step management, flow viewer, flow creation with goal selection and required input detection.

## Before Committing

```bash
# Go
cd engine && make test

# TypeScript
cd web && npm run format && npm test && npm run lint && npm run type-check
```

## Testing Requirements

- **Go**: Minimum 90% test coverage, black-box tests only
- **TypeScript**: Minimum 90% test coverage, component tests colocated

## Code Quality

- No magic numbers - use named constants
- Prefer simple solutions over abstractions
- Only make changes directly requested
- Read files before editing them

## Backward Compatibility

This project is in active development with no established user base. Do not preserve backward compatibility, avoid deprecation paths, and prefer breaking changes when they simplify the system.
