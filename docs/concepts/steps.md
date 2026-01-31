# Steps

A **step** is a unit of work with declared inputs and outputs. The engine decides when a step can run based on available data and goals.

## Step Types

Argyll supports four step types. Choose the simplest type that fits your needs:

### Sync HTTP

**Use when:** Work finishes within the HTTP request timeout and the caller can return outputs immediately.

**How it works:**
- Engine calls your HTTP endpoint with inputs and metadata
- Your handler processes the request and returns outputs
- Engine records the work completion and continues the flow

**Example:**
```json
{
  "step_id": "lookup-customer",
  "type": "sync",
  "http": {
    "endpoint": "https://api.example.com/customers/lookup",
    "timeout": 5000
  },
  "attributes": {
    "customer_id": { "required": true },
    "email": { "required": false }
  }
}
```

**Pros:** Simplest to implement, easy to debug, good for fast lookups

**Cons:** Latency bound by HTTP timeout, not for long-running work

### Async HTTP

**Use when:** Work is long-running, requires queueing, or is handled by background workers.

**How it works:**
- Engine calls your HTTP endpoint and includes a webhook URL in metadata
- Your handler returns immediately (e.g., 202 Accepted)
- Your background worker processes the task and POSTs results to the webhook

**Example:**
```json
{
  "step_id": "process-payment",
  "type": "async",
  "http": {
    "endpoint": "https://api.example.com/payments/initiate",
    "timeout": 1000
  },
  "attributes": {
    "amount": { "required": true },
    "transaction_id": { "required": false }
  }
}
```

**Pros:** Decouples execution time from request latency, supports background workers, avoids HTTP timeouts

**Cons:** Requires webhook reachability, more moving parts to operate

See [Async Steps](../guides/async-steps.md) for webhook details.

### Script

**Use for:** Small transformations, predicates, routing logic, and in-engine data processing.

**How it works:**
- Ale or Lua code runs inside the engine
- The script receives inputs and returns outputs
- No external service required

**Languages:**
- **Ale**: Purely functional, no I/O, no resource limits. Simple and fast.
- **Lua**: Limited sandboxing (io/os/debug excluded), no resource limits. More flexible.

**Example:**
```json
{
  "step_id": "calculate-discount",
  "type": "script",
  "script": {
    "language": "ale",
    "script": "(* amount (- 1 discount_percent))"
  },
  "attributes": {
    "amount": { "required": true },
    "discount_percent": { "required": true },
    "discounted_amount": { "required": false }
  }
}
```

**Pros:** No separate runtime, great for glue logic, keeps simple logic close to the flow

**Cons:** No external I/O, use sparingly for complex logic

### Flow (Sub-flow)

**Use when:** You want reusable sub-flows or shared logic across multiple flows.

**How it works:**
- Parent flow starts a child flow with its own goals
- Inputs are mapped from parent to child
- Child outputs are mapped back to parent attributes
- Child completion produces the mapped outputs

**Example:**
```json
{
  "step_id": "authorize-user",
  "type": "flow",
  "flow": {
    "goals": ["fetch-user"],
    "input_map": {
      "user_id": "uid"
    },
    "output_map": {
      "user_name": "name",
      "is_admin": "admin"
    }
  },
  "attributes": {
    "uid": { "required": true },
    "name": { "required": false },
    "admin": { "required": false }
  }
}
```

**Pros:** Reuse common patterns, encapsulate logic, allow composition without duplication

**Cons:** More events and state to manage, requires careful input/output mapping

## Inputs and Outputs

Each step declares what it needs and what it produces.

**Attributes** (in step definition):
```json
{
  "attributes": {
    "customer_id": { "required": true },
    "order_data": { "required": true },
    "notes": { "required": false },
    "confirmation_id": { "required": false },
    "total_amount": { "required": false }
  }
}
```

**Required attributes** must be available before the step executes.

**Optional attributes** have defaults (typically empty/null) if not provided.

**Produced outputs** are the attributes this step creates. When the step completes, its outputs become flow attributes available to downstream steps.

## Predicates

A step can include an optional **predicate** script that decides whether the step should execute given its inputs.

```json
{
  "step_id": "maybe-send-notification",
  "predicate": {
    "language": "ale",
    "script": "(> amount 100)"
  },
  ...
}
```

The predicate evaluates to true/false. If false, the step is skipped and produces no outputs. If true, the step executes normally.

**Use predicates for:**
- Lightweight gating logic
- Conditional execution without adding branching infrastructure
- Avoiding unnecessary work (e.g., "only notify if order is large")

**Predicates are evaluated:** Once per step before work items are created. If the predicate is false, the step produces no outputs and is marked skipped.

## Work Items and For Each

Any step can expand into multiple work items using the `for_each` attribute. When an input is marked `for_each` and provided as an array, the engine creates one work item per array element.

```json
{
  "attributes": {
    "items": { "required": true, "for_each": true },
    "item_total": { "required": false }
  }
}
```

If called with:
```json
{
  "items": [
    { "id": "item-1", "price": 10 },
    { "id": "item-2", "price": 20 },
    { "id": "item-3", "price": 15 }
  ]
}
```

The step expands to 3 work items (one per element). Each work item executes independently and produces its own outputs. When all work items complete, their outputs are aggregated back into the parent flow.

**Parallelism:** Work items can be processed with a concurrency limit. See [Work Items](../guides/work-items.md).

## Attributes vs Arguments vs Outputs

These terms describe different levels of a flow's data:

| Term | Definition | Scope |
|------|-----------|-------|
| **Attribute** | Data in the flow state, with provenance tracking | Flow lifecycle |
| **Argument** | Input values passed to a step execution | Single step execution |
| **Output** | Values returned by a step and added to flow attributes | Step result |

**Data flow:**
```
Flow attributes → Step arguments → Step logic → Step outputs → Flow attributes
```

Each attribute tracks which step produced it (provenance). This gives you a complete audit trail of how data moved through the flow.
