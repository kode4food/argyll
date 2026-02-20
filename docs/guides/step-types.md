# Choosing a Step Type

Argyll supports four step types. Choose the simplest one that fits your workload and operational constraints. For detailed information about each type, see [Steps](../concepts/steps.md).

## Quick Decision Tree

```
Does your work finish within ~10 seconds?
├─ Yes, need external service?
│  ├─ Yes → Sync HTTP (fast endpoint call)
│  └─ No → Script (in-engine transformation)
└─ No, long-running work?
   ├─ Yes → Async HTTP (queued background work)
   └─ Yes (but reusable logic) → Flow (sub-flow)
```

## Step Type Comparison

| Type | Latency | When to Use | Complexity |
|------|---------|-------------|------------|
| **Sync HTTP** | Fast (~100ms-10s) | Fast lookups, transformations | Low |
| **Async HTTP** | Decoupled (background) | Long-running, queued work | Medium |
| **Script** | In-engine (fast) | Transformations, predicates, glue logic | Low |
| **Flow** | Variable | Reusable logic, composition | High |

## Sync HTTP

**Use when:** Work finishes within the HTTP request timeout and you can return outputs immediately.

**Example:** Lookup service, calculation, validation

**Characteristics:**
- Latency bound by HTTP timeout (typically 10-30 seconds)
- Simplest to implement
- No background workers needed
- Good for: lookups, transformations, fast API calls

**Timeout:** Configure `http.timeout` per step (in milliseconds)

See [HTTP Steps](../concepts/steps.md#sync-http) for details.

## Async HTTP

**Use when:** Work is long-running, requires queueing, or is handled by background workers.

**Example:** Payment processing, email delivery, batch import

**Characteristics:**
- Decouples request latency from work duration
- Requires webhook for completion notification
- Needs background worker infrastructure
- Engine-provided receipt tokens make completion idempotent for webhook retries

See [Async Steps Guide](./async-steps.md) for webhook setup and best practices.

## Script

**Use when:** You need small transformations, predicates, or in-engine data processing.

**Example:** Calculate discount, validate email, format data

**Languages:** Ale (simple, safe) or Lua (flexible, partial sandbox)

**Characteristics:**
- No external service needed
- Runs inside the engine
- Great for glue logic
- No I/O or external calls

For predicates specifically, see [Predicates Guide](./predicates.md).

## Flow (Sub-Flow)

**Use when:** You want to reuse logic across multiple flows or encapsulate complex logic.

**Example:** User authentication flow, payment authorization, order validation

**Characteristics:**
- Child flow has its own execution plan and goals
- Input/output mapping between parent and child via attribute `mapping.name`
- More events and state overhead
- Enables composition and reuse

For details, see [Flows](../concepts/flows.md) and [Flow Steps](../concepts/steps.md#flow-sub-flow).

## Multiple Work Items (For Each)

Any step type can process multiple items using `for_each`. The engine creates one work item per array element and aggregates results.

**Example:**
```
Process 100 orders in parallel (with rate limiting)
├─ parallelism: 5 (process 5 at a time)
├─ Sync HTTP step executes per order
└─ Results aggregated back to flow
```

See [Work Items Guide](./work-items.md) for configuration and aggregation details.

## Decision Guide by Scenario

### Scenario: Fetch user from database

**Best choice:** Sync HTTP

```json
{
  "id": "lookup-user",
  "name": "Lookup User",
  "type": "sync",
  "http": {
    "endpoint": "https://api.example.com/users",
    "timeout": 5000
  },
  "attributes": {
    "user_id": { "role": "required", "type": "string" },
    "user": { "role": "output", "type": "object" }
  }
}
```

### Scenario: Process 1000 orders

**Best choice:** Sync HTTP + For Each + Parallelism

```json
{
  "id": "process-orders",
  "name": "Process Orders",
  "type": "sync",
  "http": {
    "endpoint": "https://api.example.com/orders/process",
    "timeout": 5000
  },
  "work_config": { "parallelism": 10 },
  "attributes": {
    "orders": { "role": "required", "type": "array", "for_each": true },
    "processed_order": { "role": "output", "type": "object" }
  }
}
```

### Scenario: Charge a credit card

**Best choice:** Async HTTP (because payment processing takes time and requires retry logic)

```json
{
  "id": "charge-card",
  "name": "Charge Card",
  "type": "async",
  "http": {
    "endpoint": "https://payment-service.example.com/charge",
    "timeout": 1000
  },
  "attributes": {
    "card_token": { "role": "required", "type": "string" },
    "amount": { "role": "required", "type": "number" },
    "charge_id": { "role": "output", "type": "string" }
  }
}
```

### Scenario: Apply business rule (e.g., calculate discount)

**Best choice:** Script

```json
{
  "id": "calculate-discount",
  "name": "Calculate Discount",
  "type": "script",
  "attributes": {
    "original_amount": { "role": "required", "type": "number" },
    "discount_rate": { "role": "required", "type": "number" },
    "discounted_amount": { "role": "output", "type": "number" }
  },
  "script": {
    "language": "ale",
    "script": "{:discounted_amount (* original_amount (- 1 discount_rate))}"
  }
}
```

### Scenario: Reuse authentication logic in multiple flows

**Best choice:** Flow (sub-flow)

```json
{
  "id": "authorize-user",
  "name": "Authorize User",
  "type": "flow",
  "flow": {
    "goals": ["verify-credentials"]
  },
  "attributes": {
    "user": { "role": "required", "mapping": { "name": "username" } },
    "authorized": { "role": "output", "mapping": { "name": "is_authorized" } }
  }
}
```

## Related Guides

- [Work Items](./work-items.md) - Scaling fan-out without custom code
- [Async Steps](./async-steps.md) - Webhook setup and background processing
- [Predicates](./predicates.md) - Conditional execution with scripts
- [Memoization](./memoization.md) - Result caching for expensive operations

## Key Principles

1. **Pick the simplest type** that meets your latency and operational needs
2. **Prefer Sync HTTP** for speed and simplicity
3. **Use Async HTTP** only when work is genuinely long-running
4. **Use Script** for lightweight transformations and predicates
5. **Use Flow** when you need composition and reuse
6. **Use For Each + Parallelism** for scalable fan-out without custom orchestration
