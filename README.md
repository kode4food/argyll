# Spuds <img src="./web/public/logo512.png" align="right" height="100"/>

### Goal-Oriented Workflow Engine

[![Go Report Card](https://goreportcard.com/badge/github.com/kode4food/spuds/engine)](https://goreportcard.com/report/github.com/kode4food/spuds/engine) [![Build Status](https://github.com/kode4food/spuds/workflows/Build/badge.svg)](https://github.com/kode4food/spuds/actions) [![GitHub](https://img.shields.io/github/license/kode4food/spuds)](https://github.com/kode4food/spuds/blob/main/LICENSE.md)

Spuds is a workflow engine that uses goal-oriented execution with lazy evaluation. Instead of running entire workflows, you specify what you want to achieve (one or more Goal Steps) and the engine automatically determines and executes only the minimal set of steps needed.

## Installation

```bash
# Using Docker Compose (recommended)
docker compose up

# Manual installation
go install github.com/kode4food/spuds/cmd/spuds@latest
```

## How It Works

Define steps that declare their input/output requirements. Create a workflow by specifying one or more Goal Steps. The engine automatically:

1. Walks backward from the goals to build an execution plan
2. Determines which steps are actually needed
3. Executes only those steps in dependency order
4. Completes when all goals are reached

All state changes are stored as immutable events in Redis, enabling complete audit trails, state reconstruction, and real-time event streaming.

## Features

- Three step types: HTTP sync, HTTP async, or embedded scripts (Ale/Lua)
- Real-time UI with WebSocket updates
- Automatic health checks for HTTP steps
- Step retry with configurable backoff
- Multi-instance support

## API

```bash
GET/POST   /engine/step          # Step CRUD
GET/POST   /engine/workflow      # Workflow operations
POST       /engine/plan          # Preview execution plan
GET        /engine/health        # Health checks
GET        /engine/ws            # WebSocket stream
POST       /webhook/{id}/{id}/{token}  # Async callbacks
```

See `engine/docs/engine-api.yaml` for full OpenAPI specification.

## Current Status

This is a work in progress. The basics are there, but not yet ready for production use. Use at your own risk.
