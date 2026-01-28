# Step Types

Argyll is a goal-driven orchestrator. It supports four step types. Choose the simplest one that fits your workload and operational constraints.

## Choosing a step type

Start with Sync HTTP when you can return within a normal request timeout. Move to Async HTTP when you need background processing. Use Script for small in-engine transformations or predicates. Use Flow when you want a reusable sub-flow with its own goals and mappings.

## Sync HTTP

Use when the work finishes within the HTTP request timeout and the caller can return outputs immediately.

How it works:

- Engine calls your endpoint with inputs and metadata
- Your handler returns outputs synchronously
- The engine records work completion and continues the flow

Pros:

- Simplest to implement
- Easy to debug with request/response logs
- Good fit for pure transforms or fast lookups

Cons:

- Latency is bound by the HTTP request
- Not appropriate for long-running work

## Async HTTP

Use when work is long-running, requires queueing, or is handled by background workers.

How it works:

- Engine calls your endpoint and includes a webhook URL in metadata
- Your handler returns immediately (usually 200 OK)
- Your worker POSTs the result to the webhook later

Pros:

- Decouples execution time from request latency
- Supports queues and background workers
- Avoids HTTP timeouts for long jobs

Cons:

- Requires webhook reachability and idempotent handling
- More moving parts to operate

See the async guide for details: [guides/async-steps.md](./async-steps.md).

## Script

Use for small transformations, predicates, and routing logic that does not need external I/O.

How it works:

- Ale or Lua runs inside the engine
- The script returns outputs or a predicate decision
- No external service is required

Pros:

- No separate runtime or service
- Great for glue logic and light validation
- Keeps simple logic close to the flow definition

Cons:

- No external I/O
- Use sparingly for complex logic

## Flow (Sub-flow)

Use when you want reusable sub-flows or shared logic across multiple flows.

How it works:

- Parent flow starts a child flow with its own goals
- Inputs and outputs are mapped between parent and child
- Child completion produces mapped outputs back to the parent

Pros:

- Reuse common flow patterns
- Encapsulate complex logic behind a clean interface
- Allows composition without duplicating steps

Cons:

- More events and state to manage
- Requires careful mapping of inputs and outputs

## Work items and for_each

Any step type can expand into multiple work items when an input attribute is marked for_each and provided as an array. The engine creates one work item per element and aggregates outputs when the step completes.

## Parallelism

Work items can be processed with a parallelism limit in the step work config. This lets you control concurrency and avoid overwhelming downstream systems while still using for_each expansion.

## Value recap

- Pick the simplest step type that meets your latency and operational needs
- Prefer Sync HTTP for speed, Async HTTP for durability, Script for glue, Flow for reuse
- for_each and parallelism give you scalable fan-out without custom orchestration code
