# Event Sourcing

Argyll uses **event sourcing** to record every state change as an immutable event. This gives you a complete audit trail, enables recovery without external coordination, and makes failures explainable.

## Core Idea

Instead of storing only the current state, the system records every event that changes state:

```
Started with: {}
Event: flow_started
Event: step_completed (lookup_customer) → customer_name: "Alice"
Event: step_completed (calculate_total) → amount: 150.00
Event: step_completed (process_payment) → confirmation_id: "txn-12345"
Event: flow_completed

Current state: Reconstructed by replaying these events in order
```

Any flow state can be exactly reconstructed by replaying its event log. This is the guarantee: **the complete history IS the complete truth**.

## Event Recording is Unconditional

A critical design decision: **All events are recorded, regardless of flow state.**

Even after a flow fails, work items started before failure are still recorded. This matters because:

1. **Audit trail**: The complete history includes all execution attempts
2. **Idempotency**: Late-arriving work completions are still recorded for the record

```
Example: Order processing fails

Timeline:
1. reserve_inventory started
2. process_payment fails → flow_failed
3. Flow is terminal, no new steps start
4. reserve_inventory still processing...
5. reserve_inventory completes → work_succeeded event recorded
6. Outputs available in the event log for audit purposes
```

## Event Categories

### Step Registry Events

These track changes to registered steps:

- **step_registered**: Step added to registry
- **step_unregistered**: Step deleted from registry
- **step_updated**: Step definition modified
- **step_health_changed**: Step availability/health status changed

### Engine-Level Events

These affect the engine's view of active flows:

- **flow_activated**: Flow starts, added to list of active flows
- **flow_deactivated**: Flow is terminal + no active work, removed from active list
- **flow_archiving**: Deactivated flow selected for archiving to external storage
- **flow_archived**: Flow successfully archived

### Flow-Level Events

These affect the flow's execution state:

- **flow_started**: Flow begins, execution plan created
- **flow_completed**: All goal steps satisfied
- **flow_failed**: Goal step failed or became unreachable
- **step_started**: Step preparing to execute
- **step_completed**: Step finished successfully
- **step_failed**: Step encountered an error
- **step_skipped**: Predicate evaluated to false
- **work_started**: Individual work item begins execution
- **work_succeeded**: Work item completed successfully (part of a for_each expansion)
- **work_failed**: Work item failed
- **work_not_completed**: Work item reports not yet complete (triggers retry scheduling)
- **retry_scheduled**: Work item retry scheduled for future time
- **attribute_set**: Step outputs added to flow state
- **flow_digest_updated**: Flow status digest updated (internal event)

## Event Flow Diagram

```
User requests: POST /flows
         ↓
flow_started event recorded
         ↓
Engine processes event
         ↓
Ready steps execute in parallel
         ↓
step_completed → attribute_set events
         ↓
Engine checks: are all goals satisfied?
         ↓
No: loop to execute next ready steps
Yes: flow_completed event
         ↓
Work items may still be in-flight...
         ↓
work_succeeded/work_failed events recorded
         ↓
When no active work remains:
flow_deactivated event
         ↓
Archive worker evaluates flow
         ↓
flow_archiving event + archive flow starts
```

## Terminal vs Deactivated

This distinction is crucial:

- **Terminal**: The flow's goal is achieved or impossible (flow_completed or flow_failed)
  - No new steps will start
  - In-flight work may still complete and be recorded

- **Deactivated**: The flow is terminal AND no active work remains
  - All work is accounted for
  - Eligible for archiving
  - Event log is complete

## Benefits

### Auditability

Every state change is recorded with timestamp and causality:

```
2025-01-30T15:23:45Z: flow_started
2025-01-30T15:23:46Z: step_completed (lookup_customer)
2025-01-30T15:23:47Z: attribute_set (customer_name: "Alice")
2025-01-30T15:23:48Z: step_failed (process_payment, "insufficient funds")
2025-01-30T15:23:48Z: flow_failed
```

### Recovery

No external coordination needed. Replay the events:

```go
// Pseudocode
state := NewFlowState()
for event := range flow.Events {
    state = state.Apply(event)
}
// state is now the exact current state
```

### Debugging

See exactly what happened:

- Which step produced this attribute?
- Why did a step fail?
- What work was attempted?
- When did the flow change state?

### Multi-Instance Coordination

All engine instances receive all events. No locks, no leaders:

```
Instance A processes: flow_started
Instance B processes: reserved inventory step
Instance C processes: payment step

Optimistic concurrency prevents duplicates. Any instance can pick up work.
```

## How Recovery Works

1. Engine starts, loads previous state from Redis
2. If crashed mid-flow, the flow is still in the event log
3. Replay events to reconstruct state
4. Resume from where it left off
5. No external coordination needed—the event log is the source of truth

## What This Means For You

- **Debugging**: Read the event log to understand what happened
- **Auditing**: Complete trail of all state changes
- **Recovery**: System can restart and resume without external help
- **Scalability**: Multiple instances all reading from the same event log

You don't need to write state machines or recovery logic—it's built in.
