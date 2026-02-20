# Event Sourcing Implementation

## Architecture

**Storage:** Valkey backend via timebox.Store (NOT EventHub)
- Catalog state: `timebox.Store` with `timebox.Executor[*api.CatalogState]` (TrimEvents: true, snapshotted on shutdown)
- Partition state: `timebox.Store` with `timebox.Executor[*api.PartitionState]` (TrimEvents: true, snapshotted on shutdown)
- Flow state: `timebox.Store` with `timebox.Executor[*api.FlowState]` (TrimEvents: false, full event history)
- Concurrency: Optimistic (sequence-based versioning, automatic retry on conflict)

**WebSocket Notifications (separate from event sourcing):**
- EventHub: `pkg/events/hub.go` (uses `github.com/kode4food/caravan` Topic)
- Purpose: Broadcast events to WebSocket subscribers
- NOT used for event sourcing - only for real-time UI updates
- Produces from timebox events, doesn't drive execution

## State Mutations: Executor Pattern (Only Pattern)

Every state change MUST use this pattern. No exceptions.

**Core pattern:**
```go
cmd := func(state *StateType, ag *Aggregator) error {
    // 1. Read state (read-only)
    // 2. Decide if mutation needed
    // 3. Raise events via aggregator ONLY
    events.Raise(ag, eventType, eventData)

    // 4. Register side effects via ag.OnSuccess()
    // This is called INSIDE the command, runs AFTER commit
    ag.OnSuccess(func(state *StateType) {
        // Network calls, starting work, cross-aggregate ops
        // Runs ONCE after Exec completes and commits
    })

    return nil
}
executor.Exec(ctx, aggregateID, cmd)
```

**CRITICAL: Side Effects Must Use ag.OnSuccess()**

Anything with side effects (network calls, starting real work, cross-aggregate operations, retry queue updates) MUST be registered via `ag.OnSuccess()` inside the command callback, NOT called directly in the command.

**Why:** If the command retries (optimistic concurrency conflict), direct side effects execute multiple times. `ag.OnSuccess()` runs only once, after Exec commits.

**Real patterns from the codebase:**

For partition state mutations (execPartition - no side effects needed):
```go
// registry.go: UpdateStepHealth
func (e *Engine) UpdateStepHealth(stepID api.StepID, health api.HealthStatus, errMsg string) error {
    cmd := func(st *api.PartitionState, ag *PartitionAggregator) error {
        if stepHealth, ok := st.Health[stepID]; ok {
            if stepHealth.Status == health && stepHealth.Error == errMsg {
                return nil  // Idempotent
            }
        }
        return events.Raise(ag, api.EventTypeStepHealthChanged,
            api.StepHealthChangedEvent{StepID: stepID, Status: health, Error: errMsg})
    }
    _, err := e.execPartition(cmd)  // Pure mutation, no OnSuccess needed
    return err
}
```

For flow state mutations (flowTx wrapper with OnSuccess for side effects):
```go
// engine.go: StartFlow using flowTx
func (e *Engine) StartFlow(flowID api.FlowID, plan *api.ExecutionPlan, initState api.Args, meta api.Metadata) error {
    return e.flowTx(flowID, func(tx *flowTx) error {  // flowTx wraps execFlow
        if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowStarted, ...); err != nil {
            return err
        }
        tx.OnSuccess(func(*api.FlowState) {
            e.handleFlowActivated(flowID, meta)  // Side effect after commit
        })
        // Prepare initial steps (may register more OnSuccess)
        for _, stepID := range tx.findInitialSteps(tx.Value()) {
            tx.prepareStep(stepID)
        }
        return nil  // All events committed, then all OnSuccess handlers run
    })
}
```

Work execution (inside flowTx prepareStep):
```go
// flow-exec.go: prepareStep (called inside flowTx command)
func (tx *flowTx) prepareStep(stepID api.StepID) error {
    if err := events.Raise(tx.FlowAggregator, api.EventTypeStepStarted, ...); err != nil {
        return err
    }
    started, err := tx.startPendingWork(stepID, step)
    if len(started) > 0 {
        tx.OnSuccess(func(flow *api.FlowState) {
            // Execute work AFTER commit succeeds
            tx.handleWorkItemsExecution(stepID, step, inputs, flow.Metadata, started)
        })
    }
    return nil
}
```

Flow completion (inside flowTx checkTerminal):
```go
// flow-exec.go: checkTerminal (called inside flowTx command)
func (tx *flowTx) checkTerminal() error {
    if tx.isFlowComplete(flow) {
        if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowCompleted, ...); err != nil {
            return err
        }
        tx.OnSuccess(func(*api.FlowState) {
            tx.handleFlowCompleted()  // Cleanup: remove from retry queue
        })
    }
    return nil
}
```

**Rules (non-negotiable):**
- State parameter: READ-ONLY (never modify directly)
- Mutations: ONLY via `events.Raise(ag, type, data)`
- Idempotency: Check state before raising event (avoid duplicate events)
- Executor: Handles conflict retries automatically (no manual retry needed)
- Atomicity: Events and projections commit together

## CRITICAL: Stale References After Event Raise

**MUST KNOW:** When you raise an event, the aggregator IMMEDIATELY applies it. But your state reference becomes STALE.

```go
cmd := func(st *api.FlowState, ag *FlowAggregator) error {
    // st is current state
    flow := ag.Value()  // Same as st

    // Raise event: aggregator IMMEDIATELY applies it
    if err := events.Raise(ag, api.EventTypeAttributeSet,
        api.AttributeSetEvent{...}); err != nil {
        return err
    }

    // ❌ WRONG: flow is now stale! It doesn't have the new attribute
    if v, ok := flow.GetAttributes()[name]; ok { ... }  // STALE DATA

    // ✅ CORRECT: fetch fresh state after raising event
    updatedFlow := ag.Value()  // NEW reference with applied event
    if v, ok := updatedFlow.GetAttributes()[name]; ok { ... }  // FRESH
}
```

**Why:** Persistent data structures mean events create new aggregate versions. Old references point to old versions.

**Pattern when chaining operations:**
```go
cmd := func(st *api.FlowState, ag *FlowAggregator) error {
    // Raise event 1
    if err := events.Raise(ag, EventType1, data1); err != nil {
        return err
    }
    // Fetch updated state
    current := ag.Value()

    // Check updated state and maybe raise event 2
    if someCondition(current) {
        if err := events.Raise(ag, EventType2, data2); err != nil {
            return err
        }
    }
    // Fetch again if you need latest
    latest := ag.Value()
    // ... use latest

    return nil
}
```

**Key locations to check:**
- `engine/internal/engine/flow-exec.go` - prepareStep, handleWorkSucceeded (chains events)
- Anywhere you raise multiple events in sequence

## Critical Constraint: Event Recording vs Step Launching

**MAINTAIN THIS SEPARATION:**
```go
// In flowTx (flow execution context):
// 1. Always record event (even if flow is terminal)
events.Raise(ag, api.EventTypeWorkSucceeded, ...)

// 2. SEPARATE decision: only execute next steps if flow is active
if !isTerminal(flow.Status) {
    // prepare next step
}
```

**Rationale:**
- Events recorded even after flow fails (complete audit trail)
- Step launching stopped only when flow is terminal
- Preserves complete audit trail and late-arriving work completions

Example:
```
1. Payment fails → flow_failed event
2. Flow is terminal, no new steps start
3. Inventory reservation still running → work_succeeded recorded
4. Outputs recorded in event log for complete audit trail
```

## Flow Execution Flow

```
POST /engine/flow (server)
  ↓
engine.StartFlow() calls flowTx
  ↓
flowTx raises FlowStartedEvent
  ↓
flowTx.execFlow() uses Executor pattern
  ↓
Command execution:
  - Ready steps identified
  - StepStarted event raised for each
  - Work items created and executed
  ↓
Work completes (sync) or callback received (async)
  ↓
CompleteWork() calls flowTx
  ↓
flowTx raises WorkSucceededEvent
  ↓
Aggregator updates flow state from events
  ↓
Check if all goals satisfied
  ↓
Raise FlowCompletedEvent or continue loop
  ↓
When no active work: FlowDeactivatedEvent
```

## State Reconstruction (Recovery)

Executor automatically reconstructs state by replaying events:

```go
// Conceptual - timebox handles this internally
func reconstructState(aggregateID ID) State {
    events := store.LoadEvents(aggregateID)
    state := NewState()
    for _, event := range events {
        state = applyEvent(state, event)
    }
    return state
}
```

**How recovery works:**
1. Engine.Start() calls RecoverFlows()
2. Executor loads events from Valkey for each flow
3. Replays events to reconstruct exact state
4. Resume from where it left off
5. No external coordination needed

## Event Types

**Catalog aggregate events** (step registry):
- `step_registered` - Step added to registry
- `step_unregistered` - Step deleted from registry
- `step_updated` - Step definition modified

**Partition aggregate events** (health and flow lifecycle tracking):
- `step_health_changed` - Step availability changed
- `flow_activated` - Flow added to active flows list
- `flow_deactivated` - Flow terminal + no active work
- `flow_archiving` - Flow selected for archiving
- `flow_archived` - Flow moved to external storage
- `flow_digest_updated` - Flow status digest updated (internal)

**Flow aggregate events** (flow execution state):
- `flow_started` - Execution begins
- `flow_completed` - All goals satisfied
- `flow_failed` - Goal unreachable or failed
- `step_started` - Step preparing to execute
- `step_completed` - Step succeeded
- `step_failed` - Step encountered error
- `step_skipped` - Predicate returned false
- `work_started` - Work item execution begins
- `work_succeeded` - Work item completed successfully
- `work_failed` - Work item failed
- `work_not_completed` - Work item reports not ready (triggers retry scheduling)
- `retry_scheduled` - Work item retry scheduled for future time
- `attribute_set` - Step outputs added to flow state

## Key Locations

- State mutation: `engine/internal/engine/flow-exec.go` (flowTx pattern)
- Executor setup: `engine/internal/engine/engine.go` (NewExecutor calls)
- Event types: `engine/pkg/events/` (event definitions)
- Recovery: `engine/internal/engine/recover.go` (RecoverFlows logic)
- Retry queue: `engine/internal/engine/retry_queue.go` (scheduled retries, NOT event-driven)
- WebSocket broadcast: `engine/pkg/events/hub.go` (separate from event sourcing)
