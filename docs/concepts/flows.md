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

At this point, **no new steps will start**. The flow is logically "done."

However, **work may still be in-flight**: steps started before the flow became terminal may continue executing and reporting results.

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

This separation is intentional. The flow's logical outcome (success/failure) is separate from the physical completion of all work. This enables:

- **Complete audit trail**: All step executions are recorded
- **Idempotency**: No wasted work if a step arrives late

### 3. Deactivated

The flow is deactivated once it is **terminal AND no active work remains**.

At this point:
- The flow is truly complete
- All step executions are recorded
- The flow is eligible for archiving
- No further state changes will occur

**Important:** A deactivated flow can still be read and replayed. Its event log is complete.

## Terminal State vs Deactivation

This distinction is critical:

| State | Meaning | New Steps Start? | Work In-Flight? |
|-------|---------|------------------|-----------------|
| Active | Flow is running | Yes | Maybe |
| Terminal (Completed/Failed) | Goal step finished or failed | No | Possibly |
| Deactivated | Terminal + no active work | No | No |

## Flow Attributes

**Attributes** are the data accumulated in a flow. When a step completes, its outputs become attributes.

Each attribute has **provenance**—it tracks which step produced it. This gives you a complete audit trail:

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
- **Compliance**: Complete audit trail, tamper-evident

## Archiving

Deactivated flows are eligible for archiving. The archive worker:

1. Monitors memory pressure
2. Selects deactivated flows by age and size
3. Writes them to external storage (e.g., S3)
4. Removes them from the active store

Archived flows are removed from the live engine stores. They remain available in the archive backend, but are not returned by the engine flow query endpoints.

See [Configuration](../guides/configuration.md) for archiving policy options.
