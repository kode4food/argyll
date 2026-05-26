# Core Concepts

This directory contains the foundational concepts you need to understand Argyll. Start with **execution.md**, then explore specific topics as needed.

- [Execution](./execution.md) - Goal-driven execution, lazy evaluation
- [Steps](./steps.md) - Step types, inputs, outputs, and attributes
- [Flows](./flows.md) - Flow lifecycle, terminal states, deactivation
- [Event Sourcing](./event-sourcing.md) - How the system records and recovers state
- [Architecture](./architecture.md) - Design principles and non-goals

**Quick answers:**

- **What is a goal?** The target step(s) that define "done" for a flow. The engine starts from the goals and finds the minimal set of steps needed.

- **What are attributes?** Data flowing through a flow. An upstream step produces values used by a downstream step; when a step completes, its outputs become attributes available to those downstream steps.

- **What happens to my flow after it completes?** Callers can use a completed or failed outcome immediately, but work that has already started may still produce side effects (changes to external systems) and compensation may still run. The engine deactivates the flow once no pending, active, or compensating work remains.

- **Is my data safe?** Yes. All state changes are recorded as immutable events in the shared Timebox store. You can replay the event log to reconstruct any flow state exactly.

## Terminology

- **Side effect**: A change outside flow state, such as charging a card, sending an email, or updating a database.
- **Idempotency / idempotent**: A request is idempotent when repeating it does not repeat its side effects. Use the receipt token as the idempotency key for external work.
- **Predicate**: A condition script that decides whether a step should run.
- **Memoization / memoizable**: Reusing a cached successful result for the same inputs instead of running the step again.
- **Aggregate**: The event stream for one resource, such as the catalog, the cluster, or one flow.
- **Projection**: Current state produced by applying an aggregate's events in order.
- **Upstream / downstream**: An upstream step produces a value; a downstream step consumes that value later in the flow.
- **Fan-out / fan-in**: Fan-out creates multiple work items from input values; fan-in combines results for later work.
- **Backoff**: The delay before another retry attempt.
- **Terminal**: A flow outcome is decided (`completed` or `failed`), so no new steps start, although previously started work may still produce side effects.
- **Deactivated**: A terminal flow with no pending, active, or compensating work remaining; no more side effects can be produced by that flow.
