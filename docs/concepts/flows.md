# Flows

A **flow** is a single execution of a goal-driven plan. You start a flow by specifying goals and initial inputs. The engine computes the execution plan and runs it.

## Flow Lifecycle

### 1. Active

The flow is executing. Steps are running, events are being recorded, and the flow progresses toward its goals.

**What happens:**
- Engine receives the flow start request
- Execution plan is computed and validated
- Steps execute in dependency order
- Each step completion produces attributes available to downstream steps
- Flow remains active until a goal step fails or all goals are satisfied

### 2. Terminal State: Completed or Failed

The flow reaches a terminal state when:

- **Completed**: All goal steps have produced their outputs
- **Failed**: A goal step failed or became unreachable (a required dependency failed)

At this point, **no new steps will start** and callers can use the outcome immediately.

However, **work that has already started may still be running**: steps started before the flow became terminal may still produce **side effects** (changes to external systems) and report results. Compensation may also run after a failed outcome.

```
Example: Order processing
1. Payment processing starts
2. Inventory reservation starts
3. Payment fails → flow_failed event
4. No new steps start
5. Inventory reservation still running...
6. 500ms later: Inventory reservation completes
7. work_succeeded event recorded (complete audit trail maintained)
```

This separation is intentional. Callers may use the flow's success or failure outcome before remaining work has finished. A deactivated flow is one where no further work or compensation can produce side effects. This enables:

- **Complete audit trail**: All step executions are recorded
- **Complete state recording**: Results from work that finishes late are still retained

Use full flow state or `flow_completed`/`flow_failed` events to act on the outcome. Deactivation indicates that no remaining work can produce side effects; it does not indicate when callers could first use the outcome.

### 3. Deactivated

The flow is deactivated once it is **terminal AND no work remains that can still produce side effects**: no pending, active, or compensating work items remain.

At this point:
- No remaining work can produce side effects
- All step executions are recorded
- No further work or compensation can produce side effects

**Important:** A deactivated flow can still be read and replayed. Its event log is complete.

## Terminal State vs Deactivation

This distinction is critical:

| State | Meaning | New Steps Start? | Work Already Running? |
|-------|---------|------------------|-----------------|
| Active | Flow is running | Yes | Maybe |
| Terminal (Completed/Failed) | Outcome callers can use immediately; goal step finished or failed | No | Possibly |
| Deactivated | Terminal + no pending, active, or compensating work | No | No |

## Flow Attributes

**Attributes** are the data accumulated in a flow. When a step completes, its outputs become attributes.

Each attribute has **provenance**, meaning it records which step produced it. This gives you a complete audit trail:

```json
{
  "customer_name": {
    "value": "Alice",
    "step": "lookup_customer"
  },
  "amount": {
    "value": 150.00,
    "step": "calculate_total"
  },
  "confirmation_id": {
    "value": "txn-12345",
    "step": "process_payment"
  }
}
```

Provenance answers: "Where did this data come from?"

## Flow State Reconstruction

Because all state changes are recorded as events, you can **replay** any flow's event log to reconstruct its exact state at any point in time.

This enables:

- **Debugging**: Understand exactly what happened
- **Recovery**: No external coordination needed
- **Compliance**: Complete audit trail that exposes later changes
