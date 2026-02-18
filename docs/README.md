# Documentation

Argyll is a goal-driven orchestrator: you declare goals, the engine builds an execution plan, and it performs only what matters. This documentation shows you how to use it.

## Getting Started (Read These First)

If you're new to Argyll, start here in order:

1. **[quickstart.md](./quickstart.md)** - A runnable end-to-end example (5 min read)
2. **[dev-setup.md](./dev-setup.md)** - Local development setup with Docker
3. **[concepts/index.md](./concepts/index.md)** - Core concepts and vocabulary

## Core Concepts

Understand how Argyll works:

- [Execution](./concepts/execution.md) - Goal-driven execution, lazy evaluation, execution plans
- [Steps](./concepts/steps.md) - Step types, inputs, outputs, attributes
- [Flows](./concepts/flows.md) - Flow lifecycle, terminal states, deactivation
- [Event Sourcing](./concepts/event-sourcing.md) - Complete audit trails, recovery, debugging
- [Architecture](./concepts/architecture.md) - Design principles and non-goals

## How-To Guides

Practical guides for specific tasks:

- [Choosing a Step Type](./guides/step-types.md) - Decision tree for sync vs async vs script vs flow
- [Work Items and Parallelism](./guides/work-items.md) - for_each expansion, parallelism, output aggregation
- [Async Steps and Webhooks](./guides/async-steps.md) - Background processing with webhook callbacks
- [Predicates](./guides/predicates.md) - Conditional step execution with scripts
- [Memoization](./guides/memoization.md) - Result caching for expensive operations
- [Retries and Backoff](./guides/retries.md) - Configuring retry behavior
- [Configuration](./guides/configuration.md) - Environment variables, security, deployment, monitoring
- [Flow Design Patterns](./guides/flow-design.md) - Structuring flows for reuse and clarity

## API Reference

- **Engine API**: [api/engine-api.yaml](./api/engine-api.yaml) - OpenAPI specification
- **Step Interface**: [api/step-interface.yaml](./api/step-interface.yaml) - What step handlers implement
- **Quick Reference**: [api/README.md](./api/README.md) - Curl examples and endpoint summary
- **WebSocket API**: [api/websocket.md](./api/websocket.md) - Real-time event stream for flow monitoring

## Running Examples

Complete, runnable example steps:

- [examples/README.md](../examples/README.md) - 7 example implementations (sync HTTP, async HTTP, scripts)

## Need Help?

- **Questions about concepts?** Read the [core concepts](./concepts/index.md)
- **How do I build a step?** See [Step Types Guide](./guides/step-types.md)
- **How do I deploy?** Read [Configuration Guide](./guides/configuration.md)
- **Errors or unexpected behavior?** Check the troubleshooting sections in relevant guides

## Documentation Structure

```
docs/
├── quickstart.md                    # Start here
├── dev-setup.md                     # Local development
├── concepts/
│   ├── index.md                     # Concept map
│   ├── execution.md                 # Goal-driven execution
│   ├── steps.md                     # Step types and mechanics
│   ├── flows.md                     # Flow lifecycle
│   ├── event-sourcing.md            # Event recording and recovery
│   └── architecture.md              # Design principles and boundaries
├── guides/
│   ├── step-types.md                # Choosing step types
│   ├── work-items.md                # Parallelism and fan-out
│   ├── predicates.md                # Conditional execution
│   ├── async-steps.md               # Background processing
│   ├── memoization.md               # Result caching
│   ├── retries.md                   # Retry behavior
│   ├── flow-design.md               # Flow patterns
│   └── configuration.md             # Env vars, deployment, security
├── api/
│   ├── engine-api.yaml              # OpenAPI spec
│   ├── step-interface.yaml          # Step handler interface
│   └── README.md                    # Quick reference
└── img/                             # Images and logos
```
