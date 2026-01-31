# Core Concepts

This directory contains the foundational concepts you need to understand Argyll. Start with **execution.md**, then explore specific topics as needed.

- [Execution](./execution.md) - Goal-driven execution, lazy evaluation
- [Steps](./steps.md) - Step types, inputs, outputs, and attributes
- [Flows](./flows.md) - Flow lifecycle, terminal states, deactivation
- [Event Sourcing](./event-sourcing.md) - How the system records and recovers state
- [Architecture](./architecture.md) - Design principles and non-goals

**Quick answers:**

- **What is a goal?** The target step(s) that define "done" for a flow. The engine walks backward from goals to find the minimal set of steps needed.

- **What are attributes?** Data flowing through a flow. Each step declares inputs and outputs; when complete, outputs become attributes available to downstream steps.

- **What happens to my flow after it completes?** The engine marks the flow as completed/failed, records any in-flight work completions, and deactivates it when no work remains.

- **Is my data safe?** Yes. All state changes are recorded as immutable events in Redis. You can replay the event log to reconstruct any flow state exactly.
