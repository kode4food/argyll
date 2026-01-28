# Event-Driven Architecture

## Overview

Pure event-driven architecture where flow execution is triggered by events,
not polling. Provides clear causality, better performance, horizontal scaling.

## Core Components

**EventHub (Topic-Based Distribution)**
- Location: `internal/events/hub.go`
- Uses `github.com/kode4food/caravan` Topic
- Pull-based consumers (no dropped events)
- FIFO guarantees per consumer

**Engine Event Processing Loop**
- Location: `internal/engine/engine.go`
- Event types that trigger processing:
  - `flow_started` - Initial processing
  - `step_completed` - May unblock dependent steps
  - `step_failed` - Check if flow can continue
  - `step_skipped` - May unblock dependent steps
  - `attribute_set` - May satisfy step dependencies

## Critical Design Decision

**Separate Event Recording from Step Launching**

- **All Events Are Processed**: Events recorded regardless of flow state
  (Active, Completed, or Failed). Ensures complete final state for
  compensation/reversal.

- **Single Decision Point for Step Launching**: Only `processFlow` decides
  whether to launch new steps. Returns early if flow is terminal.

**Why This Matters:**
1. Steps may complete after flow fails - outputs needed for reversal
2. Final state reflects ALL step executions
3. Clear separation: event recording (always) vs step launching (conditional)

## Event Flow

```
POST /engine/flows
  ↓
flow_started event appended
  ↓
EventHub notified
  ↓
Engine consumer receives event
  ↓
processFlow spawned
  ↓
Ready steps executed in parallel
  ↓
step_completed events → attribute_set events
  ↓
Loop until flow complete/failed
  ↓
flow_completed or flow_failed event
  ↓
Remaining work items complete (recorded for audit)
  ↓
flow_deactivated event (when terminal + no active work)
  ↓
Archive worker evaluates deactivated flows
  ↓
flow_archiving event + archive flow started
```

## Flow Lifecycle Events

**Engine-Level Events** (affect engine state):
- `flow_activated` - Emitted when flow starts, adds to active flows
- `flow_deactivated` - Emitted when flow is terminal AND no active work items
- `flow_archiving` - Emitted when a deactivated flow is selected for archiving

**Flow-Level Events** (affect flow state):
- `flow_started` - Flow begins execution
- `flow_completed` - All goals satisfied
- `flow_failed` - Goal step failed or unreachable

**Deactivation vs Completion/Failure**

`flow_completed`/`flow_failed` mark the flow's logical outcome but don't
immediately deactivate. Work items may still be in-flight:

```
1. Payment and reservation run in parallel
2. Payment fails → flow_failed emitted
3. Reservation still running, completes 100ms later
4. work_succeeded recorded (outputs available for compensation)
5. No more active work → flow_deactivated emitted
6. Archive worker selects deactivated flows
7. flow_archiving emitted and archive flow started
```

This separation enables:
- Complete audit trail of all step executions
- Outputs available for compensation/reversal
- Policy-based archiving (memory pressure, age, etc.)

## Multi-Instance Coordination

- All engines receive all flow events (broadcast)
- Optimistic concurrency prevents duplicate execution
- Any engine can pick up work (natural load balancing)
- No distributed locks required

## State Management

**Event-Sourced Storage**
- Valkey backend with atomic Lua operations
- State reconstructed from event log
- Cached projections for efficient queries
- Optimistic concurrency with sequence-based versioning

**State Properties**
- All state changes are events
- Cumulative state built from event stream
- Complete reconstruction on restart

## Consistency via timebox.Executor

All state mutations use the Executor pattern. Define a command function that
receives current state and an aggregator, then call `Exec`:

```go
// Engine state mutations
cmd := func(st *api.EngineState, ag *Aggregator) error {
    // Check current state
    if stepHealth, ok := st.Health[stepID]; ok {
        if stepHealth.Status == health {
            return nil  // No change needed
        }
    }
    // Raise event via aggregator
    return events.Raise(ag, api.EventTypeStepHealthChanged,
        api.StepHealthChangedEvent{StepID: stepID, Status: health})
}
_, err := e.engineExec.Exec(ctx, events.EngineID, cmd)

// Flow state mutations
cmd := func(st *api.FlowState, ag *FlowAggregator) error {
    exec, ok := st.Executions[stepID]
    if !ok {
        return fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
    }
    if !stepTransitions.CanTransition(exec.Status, toStatus) {
        return fmt.Errorf("%w: invalid transition", ErrInvalidTransition)
    }
    return events.Raise(ag, eventType, eventData)
}
_, err := e.flowExec.Exec(ctx, flowKey(flowID), cmd)
```

**Key Rules:**
- State is read-only inside the command - never mutate directly
- All changes via `events.Raise(ag, type, data)`
- Executor handles optimistic concurrency with automatic retry
- Events and projections update atomically on commit
