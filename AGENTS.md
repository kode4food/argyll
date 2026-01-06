# Argyll - AI Development Guide

Distributed, goal-oriented workflow processing system using lazy evaluation and
event sourcing. Steps declare input/output dependencies; the engine orchestrates
execution based on state availability, executing only the minimal set of steps
needed to reach specified goals.

## Quick Reference

### Build & Test

```bash
# Go
cd engine && make test

# TypeScript
cd web && npm run format && npm test && npm run lint && npm run type-check
```

### Docker

```bash
docker compose up                      # All services
docker compose up valkey argyll-engine # Core only
docker compose logs -f argyll-engine   # Follow logs
```

## Documentation

AI-specific development guides are in `docs/agents/`:

- **[Architecture & Design](docs/agents/architecture.md)** - Core principles, constraints, non-goals
- **[Core Concepts](docs/agents/concepts.md)** - Args vs Attributes, goal-oriented execution, step types
- **[Event Sourcing](docs/agents/event-sourcing.md)** - Event-driven architecture, flow lifecycle
- **[Deployment](docs/agents/deployment.md)** - Environment variables, store separation, production setup
- **[Go Style Guide](docs/agents/go-style.md)** - Naming conventions, testing, error handling
- **[TypeScript Style Guide](docs/agents/typescript-style.md)** - React patterns, naming, testing

Other documentation:
- **[Engine Overview](docs/engine.md)** - Detailed engine documentation
- **[API Docs](docs/api/)** - OpenAPI specifications

## Package Structure

```
engine/
  internal/
    engine/      # Core orchestration (engine.go, flow.go, flow-exec.go, step.go, work-exec.go)
    events/      # Event sourcing types and projections
    hibernate/   # Flow archival to blob storage
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

### ✅ Fully Implemented

1. Distributed coordination (optimistic concurrency)
2. Event sourcing (Valkey backend)
3. Lazy evaluation (goal-oriented execution)
4. Step types (sync HTTP, async HTTP, script)
5. Immutable execution plans
6. Real-time UI (React 19 + WebSocket)
7. Health monitoring
8. Separate engine/workflow stores
9. Step retry with configurable backoff
10. Flow hibernation (S3, GCS, Azure archival)

### ⚠️ Partial

- Script security: sandboxed but no resource limits
- Input validation: UI correct, server permissive

### ❌ Not Implemented

- Workflow pending state (immediate activation)
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
  "meta": { "workflow_id": "wf-123" }
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

Features: Dashboard, step management, workflow viewer, flow creation with
goal selection and required input detection.

## Development Workflow

### Before Committing

```bash
# Go
cd engine && make test

# TypeScript
cd web && npm run format && npm test && npm run lint && npm run type-check
```

### Testing Requirements

- **Go**: Minimum 90% test coverage, black-box tests only
- **TypeScript**: Minimum 90% test coverage, component tests colocated

### Code Quality

- Follow style guides strictly (see docs/go-style.md and docs/typescript-style.md)
- No magic numbers - use named constants
- Prefer simple solutions over abstractions
- Only make changes directly requested
- Read files before editing them
