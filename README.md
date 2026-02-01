# Argyll <img src="./web/public/argyll-logo.png" align="right" height="100"/>

### Goal-Driven Orchestrator

[![Go Report Card](https://goreportcard.com/badge/github.com/kode4food/argyll/engine)](https://goreportcard.com/report/github.com/kode4food/argyll/engine) [![Build Status](https://github.com/kode4food/argyll/workflows/Build/badge.svg)](https://github.com/kode4food/argyll/actions) [![Code Coverage](https://qlty.sh/gh/kode4food/projects/argyll/coverage.svg)](https://qlty.sh/gh/kode4food/projects/argyll) [![Maintainability](https://qlty.sh/gh/kode4food/projects/argyll/maintainability.svg)](https://qlty.sh/gh/kode4food/projects/argyll) [![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://github.com/kode4food/argyll/blob/main/LICENSE.md)

Argyll is a goal-driven orchestrator. You declare what you want to achieve, the engine builds an execution plan, and it executes only the minimal set of steps needed. All state changes are immutable events, giving you a complete audit trail.

![Argyll UI Screenshot](./docs/img/screenshot.png)

## Getting Started

```bash
# Start with Docker Compose
docker compose up
# Engine: http://localhost:8080
# UI: http://localhost:3001
```

**New to Argyll?** Start here:
1. [Quickstart](docs/quickstart.md) - 5-minute end-to-end example
2. [Core Concepts](docs/concepts/index.md) - Goals, steps, flows, events
3. [Full Documentation](docs/README.md) - Complete docs map

## How It Works

1. **Define steps** with inputs and outputs
2. **Create a flow** by specifying goal steps
3. **Engine computes** the minimal execution plan
4. **Execute and audit** - all state changes recorded as events

## Key Features

- **Event Sourcing**: Complete audit trail of all state changes
- **Lazy Evaluation**: Execute only what's needed to reach goals
- **Multi-Instance**: Horizontal scaling with optimistic concurrency
- **Real-Time UI**: WebSocket updates, live flow monitoring
- **Four Step Types**: Sync/Async HTTP, Scripts (Ale/Lua), Sub-flows
- **Built-In Retry**: Configurable backoff strategies
- **Flow Archiving**: Automatic archiving of completed flows

## Installation

```bash
# Docker Compose (all services)
docker compose up

# Go
go install github.com/kode4food/argyll/cmd/argyll@latest

# Manual local testing
export ENGINE_REDIS_ADDR=localhost:6379
go run ./cmd/argyll
```

## API Overview

Full OpenAPI spec: [docs/api/engine-api.yaml](docs/api/engine-api.yaml)

```bash
# Steps
POST   /engine/step              # Register step
GET    /engine/step              # List all steps
GET    /engine/step/:stepID      # Get step
PUT    /engine/step/:stepID      # Update step
DELETE /engine/step/:stepID      # Delete step

# Flows
POST   /engine/flow              # Start flow
GET    /engine/flow              # List flows
GET    /engine/flow/:flowID      # Get flow state
POST   /engine/plan              # Preview execution plan

# Engine & Health
GET    /engine                   # Get complete engine state
GET    /engine/health            # Get step health status
GET    /engine/ws                # WebSocket event stream
```

## Documentation

- **[Getting Started](docs/)** - Quickstart, dev setup, concepts
- **[How-To Guides](docs/guides/)** - Step types, parallelism, predicates, configuration
- **[Go SDK](sdks/go-builder/)** - Building steps in Go
- **[Python SDK](sdks/python/)** - Building steps in Python
- **[API Reference](docs/api/)** - OpenAPI specs and curl examples
- **[Examples](examples/)** - 7 runnable example steps

## Status

Work in progress. Core features stable. Not yet production-ready. Use at your own risk.
