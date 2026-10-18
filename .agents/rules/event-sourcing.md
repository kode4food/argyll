# Event Sourcing Implementation

## Architecture

**Storage:** `timebox.Store` instances backed by Raft + Pebble

- Catalog state: `timebox.Executor[api.CatalogState]` over the engine store
- Cluster state: `timebox.Executor[api.ClusterState]` over the engine store
- Flow state: `timebox.Executor[api.FlowState]` over the flow store
- Concurrency: Optimistic (sequence-based versioning, automatic retry on conflict)

**WebSocket Notifications (separate from event sourcing):**

- EventHub: `engine/internal/event.Hub`, wired through `engine/cmd/argyll/internal/app/app.go` and `engine/cmd/argyll/internal/server/websocket.go`
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

Register external effects such as network calls and retry queue updates through `ag.OnSuccess()` inside the command callback rather than calling them directly in the command. `Inline` handlers, such as scripts, run inside the command. Related aggregate changes within the same Store belong in `ag.Transaction().Exec(...)`, so they commit atomically and retry together. Child creation joins parent work start; child settlement joins parent work completion; parent deactivation joins eligible child deactivation. Catalog registration and its cluster health update also share a transaction.

**Why:** If the command retries (optimistic concurrency conflict), direct side effects execute multiple times. `ag.OnSuccess()` runs only once, after Exec commits, and receives the final aggregate state plus the successfully flushed `[]*timebox.Event`.

**Real patterns from the codebase:**

For cluster state mutations (the cluster executor with no external side effects needed):

```go
// health.go: UpdateStepHealth
func (e *Engine) UpdateStepHealth(stepID api.StepID, health api.HealthStatus, errMsg string) error {
    nid := e.LocalNodeID()
    cmd := func(st api.ClusterState, ag *ClusterAggregator) error {
        node := st.Nodes[nid]
        if h, ok := node.Health[stepID]; ok {
            if h.Status == health && h.Error == errMsg {
                return nil // Idempotent
            }
        }
        return events.Raise(ag, api.EventTypeStepHealthChanged,
            api.StepHealthChangedEvent{
                NodeID: nid,
                StepID: stepID,
                Status: health,
                Error:  errMsg,
            },
        )
    }
    _, err := e.clusterExec.Exec(events.ClusterKey, cmd)
    return err
}
```

For flow state mutations (flowTx wrapper with OnSuccess for side effects):

```go
// flow-start.go: StartPlan using flowTx
func (e *Engine) StartPlan(
    fid api.FlowID, pl *api.ExecutionPlan, apps ...flow.Applier,
) error {
    opts := flow.Defaults(apps...)
    return e.flowTx(fid, func(tx *flowTx) error {
        return tx.startPlan(pl, opts)
    })
}

func (tx *flowTx) startPlan(pl *api.ExecutionPlan, opts *flow.Options) error {
    if tx.Value().ID != "" {
        // ... match the already started flow, or return ErrFlowExists
    }
    if err := events.Raise(
        tx.FlowAggregator, api.EventTypeFlowStarted, ...,
    ); err != nil {
        return err
    }
    // Starts ready steps (which may register OnSuccess) and arms deadlines
    return tx.advanceFlow()
}
```

Work execution (inside flowTx, after the work items are started):

```go
// work-continue.go: startContinuedWork (called inside flowTx command)
func (tx *flowTx) startContinuedWork(
    sid api.StepID, st *api.Step, started api.WorkItems,
) error {
    if st.Type == api.StepTypeFlow {
        // Child flows already started inside this transaction
        return nil
    }
    tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
        // Execute work AFTER commit succeeds
        ex := fl.Executions[sid]
        tx.executeStartedWork(st, ex.Inputs, fl.Metadata, started)
    })
    return nil
}
```

Flow completion (inside flowTx checkTerminal):

```go
// flow-stop.go: checkTerminal (called inside flowTx command)
func (tx *flowTx) checkTerminal() error {
    fl := tx.Value()
    if isFlowComplete(fl) {
        if err := events.Raise(
            tx.FlowAggregator, api.EventTypeFlowCompleted, ...,
        ); err != nil {
            return err
        }
        return tx.maybeDeactivate()
    }
    // ... same shape for the failed case
    return nil
}
```

**Rules:**

- Treat the state parameter as read-only
- OnSuccess signature: `func(state, committedEvents)`
- Mutations: use `events.Raise(ag, type, data)`
- Idempotency: check state before raising an event, so each event is raised once
- Executor: Handles conflict retries automatically (no manual retry needed)
- Atomicity: Events and projections commit together
- Scheduled tasks are never cancelled. Every handler rechecks the flow's state when its task fires and does nothing when the task no longer applies

## Stale References After Event Raise

When you raise an event, the aggregator applies it immediately and your previous state reference becomes stale.

```go
cmd := func(st api.FlowState, ag *FlowAggregator) error {
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
cmd := func(st api.FlowState, ag *FlowAggregator) error {
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
POST /engine/flows (server)
  ↓
Server validates request, builds the plan, calls engine.StartFlow()
  ↓
engine.StartFlow() calls flowTx(), which joins the flow to Store.Transact()
  ↓
Inside the executor command:
  - Raise FlowStartedEvent
  - advanceFlow (see Advancing a Flow), which for each ready step:
    - If the predicate is false, raises StepSkippedEvent
    - Otherwise raises StepStartedEvent with computed work items
    - Raises WorkStartedEvent for each work item that can start now
    - Runs Inline handlers, such as scripts, settling their work here
    - Registers OnSuccess handlers for post-commit side effects
  ↓
Commit succeeds
  ↓
OnSuccess handlers run:
  - Launch newly started work items
  ↓
Work item execution branch:
  - Sync HTTP: perform work, then call CompleteWork() on success
  - Async HTTP: invoke the step and return; webhook later calls CompleteWork()/FailWork()
  - Flow step: no external execution; parent work start and child creation already committed atomically using the precomputed child plan
  ↓
Completion transaction:
  - Raise WorkSucceededEvent / WorkFailedEvent / WorkNotCompletedEvent
  - For successful work, check step completion:
    - Maybe raise AttributeSetEvent(s)
    - Raise StepCompletedEvent or StepFailedEvent
  - Maybe schedule retries
  - Once the step settles, advanceFlow
  ↓
Terminal state (checkTerminal, inside advanceFlow):
  - Raise FlowCompletedEvent or FlowFailedEvent when appropriate, settling the parent work in the same transaction
  - Raise FlowDeactivatedEvent only after the flow is terminal, no active work or compensation remains, and the parent releases it; join eligible child deactivations to the same transaction
```

## Advancing a Flow

`advanceFlow` (`engine/internal/engine/flow-continue.go`) is the single continuation. Every path that changes flow state calls it inside the same transaction: flow start, work success and failure, step skips and predicate failures, optional-input timeouts, and recovery. Never assemble its steps by hand at a call site.

It runs, in order: skip unused pending steps, fail unreachable ones, skip again (a failed goal can leave more steps unused), check for a terminal flow, start ready steps, and arm the optional-input deadlines of steps anchored during this transaction.

Readiness is computed from flow state alone. An `attribute_set` event records a value and is never a trigger. Only `collect: first` is decided by a value existing. `last`, `some`, `all`, `none`, match gates, unreachability, goals, and compensation all depend on providers reaching a terminal state, which `step_completed`, `step_failed`, and `step_skipped` record. A step raises its `attribute_set` events together with its `step_completed`, and advancing runs after both.

The collect-mode rules live in `engine/pkg/policy/collect.go`: `InputFulfilled` and `FulfilledAt` for readiness, `InputSatisfiable` for whether a pending input can still be met. The engine gathers facts from flow state and asks policy, so readiness and reachability share one table.

A deadline's anchor never moves once set, so `flowTx` keeps the state its command started from (`start`) and `scheduleTimeouts` arms only steps whose anchor appeared since then. Recovery calls `recoverTimeouts` after `advanceFlow` to arm the steps anchored before the transaction began, so each pending step is armed once.

WebSocket notifications never drive advancement. The event hub publishes after commit and is best effort, so execution decisions stay inside the flow transaction.

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

**Cluster aggregate events** (node health tracking):

- `step_health_changed` - Step availability changed

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
- `work_retry_scheduled` - Work item retry scheduled for future time
- `dispatch_deferred` - Step dispatch deferred until a node can run it
- `comp_started` - Compensation dispatched for a succeeded work item
- `comp_retry_scheduled` - Compensation retry scheduled for future time
- `comp_succeeded` - Compensation completed successfully
- `comp_failed` - Compensation permanently failed
- `flow_deactivated` - Flow terminal + no active work
- `attribute_set` - Step outputs added to flow state

## Key Locations

- State mutation examples: `engine/internal/engine/flow-start.go`, `engine/internal/engine/step-start.go`, `engine/internal/engine/work-stop.go`
- Executor setup: `engine/internal/engine/engine.go` (NewExecutor calls)
- Event types: `engine/pkg/events/` (event definitions)
- Flow advancement: `engine/internal/engine/flow-continue.go` (`advanceFlow`)
- Collect-mode policy: `engine/pkg/policy/collect.go`
- Optional-input deadlines: `engine/internal/engine/step-continue.go`
- Recovery: `engine/internal/engine/recover.go` (RecoverFlows logic) and `engine/internal/engine/reconcile.go` (`recoverFlow`)
- Retry and deferred dispatch scheduling: `engine/internal/engine/work-continue.go`, `engine/internal/engine/step-dispatch.go`, and `engine/internal/engine/scheduler/`
- Work item deadlines for in-flight attempts: `engine/internal/engine/work-deadline.go`
- WebSocket broadcast: `engine/cmd/argyll/internal/app/app.go` and `engine/cmd/argyll/internal/server/websocket.go`
