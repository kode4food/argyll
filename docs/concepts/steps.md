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
  "id": "lookup-customer",
  "name": "Lookup Customer",
  "type": "sync",
  "http": {
    "endpoint": "https://api.example.com/customers/lookup",
    "timeout": 5000
  },
  "attributes": {
    "customer_id": { "role": "required" },
    "email": { "role": "optional" }
  }
}
```

**Pros:** Simplest to implement, easy to debug, good for fast lookups

**Cons:** Latency bound by HTTP timeout, not for long-running work

### Async HTTP

**Use when:** Work is long-running, requires queueing, or is handled by background workers.

**How it works:**
- Engine calls your HTTP endpoint and includes a webhook URL in metadata
- Your handler returns immediately with a valid StepResult payload (HTTP 200)
- Your background worker processes the task and POSTs results to the webhook

**Example:**
```json
{
  "id": "process-payment",
  "name": "Process Payment",
  "type": "async",
  "http": {
    "endpoint": "https://api.example.com/payments/initiate",
    "timeout": 1000
  },
  "attributes": {
    "amount": { "role": "required" },
    "transaction_id": { "role": "output" }
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
  "id": "calculate-discount",
  "name": "Calculate Discount",
  "type": "script",
  "script": {
    "language": "ale",
    "script": "{:discounted_amount (* amount (- 1 discount_percent))}"
  },
  "attributes": {
    "amount": { "role": "required" },
    "discount_percent": { "role": "required" },
    "discounted_amount": { "role": "output" }
  }
}
```

**Pros:** No separate runtime, great for glue logic, keeps simple logic close to the flow

**Cons:** No external I/O, use sparingly for complex logic

### Flow (Sub-flow)

**Use when:** You want reusable sub-flows or shared logic across multiple flows.

**How it works:**
- Parent flow starts a child flow with its own goals
- Inputs are mapped from parent to child via attribute `mapping.name`
- Child outputs are mapped back to parent attributes via `mapping.name`
- Child completion produces the mapped outputs

**Example:**
```json
{
  "id": "authorize-user",
  "name": "Authorize User",
  "type": "flow",
  "flow": {
    "goals": ["fetch-user"]
  },
  "attributes": {
    "uid": { "role": "required", "mapping": { "name": "user_id" } },
    "name": { "role": "output", "mapping": { "name": "user_name" } },
    "admin": { "role": "output", "mapping": { "name": "is_admin" } }
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
    "customer_id": { "role": "required" },
    "order_data": { "role": "required" },
    "notes": { "role": "optional" },
    "confirmation_id": { "role": "output" },
    "total_amount": { "role": "output" }
  }
}
```

**Required attributes** must be available before the step executes.

**Optional attributes** use their declared `default` value if provided. If no default is declared and the value is missing, the input is omitted.

Optional attributes may also declare a `timeout` (milliseconds):
- `timeout: 0` means no timeout (wait for upstream providers if they exist)
- timeout starts when the first potential upstream provider for that attribute starts work
- if the timeout expires before the attribute is produced, the consuming step may proceed with its optional `default`
- this fallback default is step-local only and does not become a flow attribute

**Produced outputs** are the attributes this step creates. When the step completes, its outputs become flow attributes available to downstream steps.

## Predicates

A step can include an optional **predicate** script that decides whether the step should execute given its inputs.

```json
{
  "id": "maybe-send-notification",
  "name": "Maybe Send Notification",
  "type": "sync",
  "http": {
    "endpoint": "https://api.example.com/notify",
    "timeout": 5000
  },
  "attributes": {
    "amount": { "role": "required" },
    "notification_sent": { "role": "output" }
  },
  "predicate": {
    "language": "ale",
    "script": "(> amount 100)"
  }
}
```

The predicate evaluates to true/false. If false, the step is skipped and produces no outputs. If true, the step executes normally.

**Use predicates for:**
- Lightweight gating logic
- Conditional execution without requiring branching infrastructure
- Avoiding unnecessary work (e.g., "only notify if order is large")

**Predicates are evaluated:** before initial work starts and again when pending/retry work items are started. If predicate checks fail, that work does not run.

## Work Items and For Each

Any step can expand into multiple work items using the `for_each` attribute. When an input is marked `for_each` and provided as an array, the engine creates one work item per array element.

```json
{
  "attributes": {
    "items": { "role": "required", "type": "array", "for_each": true },
    "item_total": { "role": "output" }
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

## Attribute Mappings

Attributes can include optional **mappings** that rename parameters or transform values between the flow state and service interfaces.

### Parameter Renaming

Use `mapping.name` to map between flow state attribute names (outer names) and service parameter names (inner names):

```json
{
  "attributes": {
    "user_email": {
      "role": "required",
      "type": "string",
      "mapping": {
        "name": "email"
      }
    }
  }
}
```

**For inputs:** The flow state has `user_email`, but the service receives it as `email`.

**For outputs:** The service returns `email`, but it's stored in flow state as `user_email`.

### Value Transformation

Use `mapping.script` to transform values using JSONPath, Ale, or Lua:

```json
{
  "attributes": {
    "order_total": {
      "role": "required",
      "type": "number",
      "mapping": {
        "script": {
          "language": "jpath",
          "script": "$.data.order.total"
        }
      }
    }
  }
}
```

**For inputs:** Extract a nested value from the flow state before passing to the service.

**For outputs:** Extract a nested value from the service response before storing in flow state.

### Combined Renaming and Transformation

You can use both `name` and `script` together:

```json
{
  "attributes": {
    "customer_id": {
      "role": "required",
      "type": "string",
      "mapping": {
        "name": "customerId",
        "script": {
          "language": "jpath",
          "script": "$.user.id"
        }
      }
    }
  }
}
```

This extracts `$.user.id` from the value, then passes it to the service as `customerId`.

**Mapping rules:**
- At least one of `name` or `script` must be present
- Mappings not allowed on const attributes
- Inner names (mapping.name) must be unique within inputs, unique within outputs

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
