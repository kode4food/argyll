# Architecture & Design Philosophy

## Core Principles

1. **Understanding stays ahead of behavior**: The system only does what can be
   fully explained via event replay. No hidden state, no implicit coordination.

2. **Constraints are protective**: Each limitation exists to preserve
   correctness, predictability, or operational simplicity.

3. **Lazy evaluation is a correctness property**: Steps execute only when their
   outputs are required by declared goals. This defines "correct execution."

4. **Explicit non-goals are first-class decisions**: Features Argyll refuses to
   implement are as important as features it provides.

## Architectural Constraints

These constraints are **load-bearing** and must not be relaxed:

- **Acyclic execution only**: Execution plans form a DAG. No cycles or loops.
- **No speculative execution**: Steps never execute "just in case."
- **No bidirectional data flow**: Data flows from dependencies toward goals.
- **No flow-level timeouts**: Flows run to completion or failure.
- **No human-in-the-loop**: No waiting states or approval gates.
- **Minimal polling**: Flow progression is event-driven.
- **No distributed locks or leaders**: Optimistic concurrency only.
- **No cross-flow coordination**: Flows are fully isolated.
- **Event recording is unconditional**: All events persisted regardless of
  flow state.

## Non-Goals

Argyll is intentionally **not**:

- A general-purpose DAG engine
- A BPMN or human flow system
- A speculative or reactive orchestration framework
- A process manager with explicit lifecycle control
- A plugin framework for arbitrary execution semantics
- A scheduler (no cron, no delayed execution)
- A distributed transaction coordinator

## Recovery and Correctness

System correctness must be explainable via event replay alone:

- Any flow state can be reconstructed by replaying its event log
- No runtime state exists that isn't derived from persisted events
- Recovery requires no external coordination or distributed consensus

## Notes

- Current ListFlows filtering that hides sub-flows does so by loading each flow and checking parent metadata; this will be too slow at scale and should be replaced with a more efficient index or query strategy.
