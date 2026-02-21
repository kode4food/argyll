# Architecture Constraints

Argyll is built around core constraints that preserve correctness, predictability, and operational simplicity. These constraints are **intentional and load-bearing**—they define what Argyll is.

## Core Design Principles

1. **Understanding stays ahead of behavior**: The system only does what can be fully explained via event replay. No hidden state, no implicit coordination.

2. **Constraints are protective**: Each limitation exists to preserve correctness, predictability, or operational simplicity.

3. **Lazy evaluation is a correctness property**: Steps execute only when their outputs are required by declared goals. This defines "correct execution."

4. **Explicit non-goals are first-class decisions**: Features Argyll refuses to implement are as important as features it provides.

## Load-Bearing Constraints

### Acyclic Execution Only

Execution plans form a **DAG** (directed acyclic graph). No cycles or loops.

**Why**: Cycles would require speculative execution or runtime plan mutation, breaking determinism.

### No Speculative Execution

Steps never execute "just in case." A step executes only if its outputs are required by the goals.

**Why**: Minimizes resource consumption and failure points.

### No Bidirectional Data Flow

Data flows from dependencies toward goals. No step receives data from its dependents.

**Why**: Makes execution deterministic and makes the plan computable at flow start time.

### No Flow-Level Timeouts

Flows run to completion or failure. You cannot specify a global timeout that cancels a flow.

**Why**: Avoids cascade failures and enables controlled failure modes.

### No Human-in-the-Loop

No waiting states or approval gates. Flows cannot pause and wait for human intervention.

**Why**: Keeps the system simple and event-driven. No persistent state for paused flows.

### Event Recording is Unconditional

All events are persisted regardless of flow state (Active, Completed, or Failed).

**Why**: Ensures complete audit trail. Late-arriving work completions are still recorded for visibility into the entire execution history.

### No Distributed Locks or Leaders

Optimistic concurrency only. No leader election, no distributed locking.

**Why**: Simpler to operate. Natural load balancing. No split-brain risk.

### No Arbitrary Cross-Flow Coordination

Flows are isolated by default. The only supported cross-flow relationship is an explicit parent/child flow step. Flows cannot otherwise block, wait for, or depend on unrelated flows.

**Why**: Keeps execution predictable while still allowing explicit sub-flow composition.

### Minimal Polling

Flow progression is event-driven, not polling-based.

**Why**: Better performance, lower latency, deterministic behavior.

## What Argyll Is NOT

Argyll intentionally does not provide:

- **General-purpose DAG engine**: Not suitable for arbitrary graph topologies
- **BPMN or human process engine**: No approval gates, waiting states, or human intervention
- **Speculative or reactive orchestration**: No "if-then" execution
- **Process manager with explicit lifecycle control**: No pause/resume or manual state transition
- **Plugin framework for arbitrary execution semantics**: Limited step types by design
- **Scheduler**: No cron jobs, delayed execution, or recurring flows
- **Distributed transaction coordinator**: No ACID guarantees across steps

## Recovery and Correctness

System correctness is explainable via event replay alone:

- Any flow state can be reconstructed by replaying its event log
- No runtime state exists that isn't derived from persisted events
- Recovery requires no external coordination or distributed consensus

This is why you don't need to write recovery logic, state machines, or leader election code—it's built in.

## Design Trade-offs

These constraints enable:

| Benefit | Cost |
|---------|------|
| Deterministic execution | Cannot change plan at runtime |
| Simple recovery | No pause/resume capability |
| Event-driven scaling | No arbitrary cross-flow coordination |
| Complete audit trail | Everything is recorded (storage overhead) |
| No distributed consensus | No human-in-the-loop |

If your use case requires runtime plan mutation, arbitrary cross-flow synchronization, or human approval gates, Argyll may not be the right fit. Consider a general-purpose orchestrator instead.

## Implications for Users

### Do This ✓

- Declare all step dependencies upfront
- Use predicates for lightweight conditional logic
- Structure flows as pure DAGs
- Compose flows with sub-flows (flow steps)

### Don't Do This ✗

- Try to add steps dynamically during flow execution
- Expect to pause/resume or seek human approval
- Rely on implicit cross-flow coordination or signaling
- Use Argyll as a general-purpose graph engine

## Operational Constraints

- **Scripts**: Ale (no I/O, purely functional) and Lua (partial sandbox, no resource limits)
- **No built-in monitoring**: Integrate with external APM tools
- **Input validation**: UI validates, server is permissive
- **Authentication**: Use reverse proxy for auth/authz

See [Configuration](../guides/configuration.md) for operational details.
