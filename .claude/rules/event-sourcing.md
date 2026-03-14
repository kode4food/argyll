# Event Sourcing Implementation

## Architecture

**Storage:** shared `timebox.Store` backed by Raft + Pebble

- Catalog state: `timebox.Executor[*api.CatalogState]` over the shared store
- Partition state: `timebox.Executor[*api.PartitionState]` over the shared store
- Flow state: `timebox.Executor[*api.FlowState]` over the shared store
- Concurrency: Optimistic (sequence-based versioning, automatic retry on conflict)

**WebSocket Notifications (separate from event sourcing):**

- EventHub: `internal/event.Hub`, wired through `cmd/argyll/main.go` and `internal/server/websocket.go`
- Purpose: Broadcast events to WebSocket subscribers
- Separate from event sourcing; used for real-time UI updates
- Produces from timebox events, doesn't drive execution

## State Mutations: Executor Pattern

State changes use this pattern.

**Core pattern:**

```go
cmd := func(state *StateType, ag *Aggregator) error {
    // 1. Read state
    // 2. Decide if mutation needed
    // 3. Raise events via the aggregator
    events.Raise(ag, eventType, eventData)

    // 4. Register side effects via ag.OnSuccess()
    ag.OnSuccess(func(state *StateType, committed []*timebox.Event) {
        // Network calls, starting work, cross-aggregate ops
    })

    return nil
}
executor.Exec(ctx, aggregateID, cmd)
```

**Side effects use `ag.OnSuccess()`**

Register side effects such as network calls, starting work, cross-aggregate operations, and retry queue updates through `ag.OnSuccess()` inside the command callback rather than calling them directly in the command.

**Why:** If the command retries (optimistic concurrency conflict), direct side effects execute multiple times. `ag.OnSuccess()` runs only once, after Exec commits, and receives the final aggregate state plus the successfully flushed `[]*timebox.Event`.

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
    _, err := e.execPartition(cmd)
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
        tx.OnSuccess(func(*api.FlowState, []*timebox.Event) {
            e.handleFlowActivated(flowID, meta)  // Side effect after commit
        })
        // Prepare initial steps (may register more OnSuccess)
        for _, stepID := range tx.findInitialSteps(tx.Value()) {
            tx.prepareStep(stepID)
        }
        return nil
    })
}
```

Work execution (inside flowTx prepareStep):

```go
// step-start.go: prepareStep (called inside flowTx command)
func (tx *flowTx) prepareStep(stepID api.StepID) error {
    if err := events.Raise(tx.FlowAggregator, api.EventTypeStepStarted, ...); err != nil {
        return err
    }
    started, err := tx.startPendingWork(stepID, step)
    if len(started) > 0 {
        tx.OnSuccess(func(flow *api.FlowState, _ []*timebox.Event) {
            // Execute work AFTER commit succeeds
            tx.handleWorkItemsExecution(stepID, step, inputs, flow.Metadata, started)
        })
    }
    return nil
}
```

Flow completion (inside flowTx checkTerminal):

```go
// flow-stop.go: checkTerminal (called inside flowTx command)
func (tx *flowTx) checkTerminal() error {
    if tx.isFlowComplete(flow) {
        if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowCompleted, ...); err != nil {
            return err
        }
        tx.OnSuccess(func(*api.FlowState, []*timebox.Event) {
            tx.handleFlowCompleted()  // Cleanup: remove from retry queue
        })
    }
    return nil
}
```

**Rules:**

- Do not mutate the state parameter directly
- OnSuccess signature: `func(state, committedEvents)`
- Mutations: use `events.Raise(ag, type, data)`
- Idempotency: Check state before raising event (avoid duplicate events)
- Executor: Handles conflict retries automatically (no manual retry needed)
- Atomicity: Events and projections commit together

## Stale References After Event Raise

When you raise an event, the aggregator applies it immediately and your previous state reference becomes stale.

```go
cmd := func(st *api.FlowState, ag *FlowAggregator) error {
    // st is current state
    flow := ag.Value()

    // Raise event: the aggregator applies it immediately
    if err := events.Raise(ag, api.EventTypeAttributeSet,
        api.AttributeSetEvent{...}); err != nil {
        return err
    }

    // flow is now stale; it doesn't have the new attribute
    if v, ok := flow.GetAttributes()[name]; ok { ... }

    // Fetch fresh state after raising the event
    updatedFlow := ag.Value()
    if v, ok := updatedFlow.GetAttributes()[name]; ok { ... }
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

- `engine/internal/engine/step-start.go`, `engine/internal/engine/work-stop.go`, `engine/internal/engine/flow-stop.go`
- Anywhere you raise multiple events in sequence

## Event Recording vs Step Launching

Keep this separation:

```go
// In flowTx (flow execution context):
// 1. Record the event, even if the flow is terminal
events.Raise(ag, api.EventTypeWorkSucceeded, ...)

// 2. Separately decide whether to execute next steps
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

```text
POST /engine/flow (server)
  ↓
Server validates request, builds the plan, calls engine.StartFlow()
  ↓
engine.StartFlow() calls flowTx(), which wraps execFlow()
  ↓
Inside the executor command:
  - Raise FlowStartedEvent
  - Find ready pending steps
  - For each ready step:
    - If the predicate is false, raise StepSkippedEvent
    - Otherwise raise StepStartedEvent with computed work items
    - Raise WorkStartedEvent for each work item that can start now
    - Register OnSuccess handlers for post-commit side effects
  ↓
Commit succeeds
  ↓
OnSuccess handlers run:
  - Schedule flow/step timeout tasks
  - Launch newly started work items
  ↓
Work item execution branch:
  - Script/sync HTTP: perform work, then call CompleteWork() on success
  - Async HTTP: invoke the step and return; webhook later calls CompleteWork()/FailWork()
  - Flow step: StartChildFlow(); parent work completes later when the child flow deactivates
  ↓
Completion transaction:
  - Raise WorkSucceededEvent / WorkFailedEvent / WorkNotCompletedEvent
  - For successful work, check step completion:
    - Maybe raise AttributeSetEvent(s)
    - Raise StepCompletedEvent or StepFailedEvent
  - Maybe schedule retries
  - Maybe skip unused pending steps
  - Maybe start newly ready pending steps
  ↓
Check terminal state:
  - Raise FlowCompletedEvent or FlowFailedEvent when appropriate
  - Raise FlowDeactivatedEvent only after the flow is terminal and no active work remains
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
2. Executor loads events from the shared Timebox store for each flow
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

- State mutation examples: `engine/internal/engine/flow-start.go`, `engine/internal/engine/step-start.go`, `engine/internal/engine/work-stop.go`
- Executor setup: `engine/internal/engine/engine.go` (NewExecutor calls)
- Event types: `engine/pkg/events/` (event definitions)
- Recovery: `engine/internal/engine/recover.go` (RecoverFlows logic)
- Retry scheduling: `engine/internal/engine/work-continue.go` and `engine/internal/engine/scheduler/`
- WebSocket broadcast: `cmd/argyll/main.go` and `internal/server/websocket.go`
