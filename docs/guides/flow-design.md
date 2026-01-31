# Flow Design Patterns

This guide covers practical patterns for designing flows effectively.

## Start with Goals

Define clear goal steps - these are the outcomes you actually need. The engine builds the minimal execution plan to reach them.

**Why:** Avoids speculative work and keeps flows focused.

**Example:**
```
Goal: process_payment + send_confirmation
NOT Goal: validate_customer, lookup_inventory (these are dependencies)

Engine computes:
- Required for process_payment: validate_customer, lookup_account
- Required for send_confirmation: process_payment (dependency)
- Executes: validate_customer → lookup_account → process_payment → send_confirmation
```

## Optional Inputs and Defaults

Use optional attributes with defaults to keep steps reusable across different flow contexts.

```json
{
  "attributes": {
    "customer_id": { "role": "required" },
    "notify_user": { "role": "optional", "default": "true" },
    "timeout_seconds": { "role": "optional", "default": "30" }
  }
}
```

**Why:** Enables the same step to work in multiple flows without duplication.

## For Each and Parallelism

Use `for_each` on array inputs to process multiple items in parallel. Outputs are aggregated.

```json
{
  "attributes": {
    "order_items": { "role": "required", "for_each": true },
    "stock_reserved": { "role": "output" }
  },
  "work_config": {
    "parallelism": 5
  }
}
```

**Why:** Enables scalable fan-out without custom orchestration code.

**Anti-pattern:** Don't use for_each if you need results back per-item before the step completes. Outputs are only available after ALL work items finish.

## Predicates for Business Logic

Use predicates to conditionally skip steps based on flow state. This avoids branching logic.

```json
{
  "id": "apply_vip_discount",
  "predicate": {
    "language": "ale",
    "script": "(and (> order_total 1000) (= customer_tier \"gold\"))"
  }
}
```

**Why:** Lightweight gating without adding extra steps or complexity.

**Pattern:** Use predicates for 1-2 condition checks. For complex logic, create dedicated decision steps.

## Error Handling: Fail-Fast vs Best-Effort

- **Goal steps**: Failure terminates the flow (fail-fast)
- **Non-goal steps**: Failure doesn't stop the flow, but downstream steps may be unreachable

```
Example Flow:
├── Step A (Goal)
├── Step B (Goal)
├── Step C (depends on B, but not a goal)
└── Step D (depends on A)

If B fails:
  - Flow fails (B is a goal)
  - A still executes
  - C is unreachable (depends on failed B)
  - D still executes (depends on A, not B)
```

**Design pattern:** Use required/optional downstream steps intentionally.

## Sub-flows for Composition

Use flow steps (sub-flows) to create reusable, self-contained units with their own goals.

```json
{
  "id": "authorize_payment",
  "type": "flow",
  "flow": {
    "goals": ["validate_payment_method"],
    "input_map": {
      "card_number": "cardnum",
      "cvv": "security_code"
    },
    "output_map": {
      "is_valid": "authorization_result"
    }
  }
}
```

**Why:** Encapsulation and reuse. The sub-flow has its own execution plan and goals.

**Pattern:** Create sub-flows for domain-specific logic (authentication, payment, validation) that might be reused across multiple parent flows.

## Idempotency Patterns

Design flows so they can be safely retried without unintended duplication.

**Pattern 1: Unique IDs for Async Work**
```
User provides: order_id = "ord-123"
Step generates: payment_request_id = "uuid()" (unique each attempt)
Problem: Different UUIDs each retry = duplicate payment requests

Better: payment_request_id = hash(order_id + "payment_v1")
Effect: Same input = same ID = idempotent at payment processor
```

**Pattern 2: Async Webhook Idempotency**
Your async worker receives the same `receipt_token` on retry. Use it to detect duplicates and return the same result.

See [Async Steps Guide](./async-steps.md) for details.

## Minimize Cross-Cutting Concerns

Avoid flows where many steps depend on each other in complex ways. Instead:
- **Keep dependencies linear when possible**: A → B → C
- **Use sub-flows** for clusters of related steps
- **Use predicates** instead of explicit branching

This makes flows easier to understand, test, and debug.

## Performance Patterns

### Large Fan-Out
```
thousands of items → for_each step with parallelism: 10
Problem: 10,000 items × timeout may exhaust connections
Better: Increase parallelism but batch in upstream step
```

### Sequential vs Parallel
```
Payment → Inventory → Shipping (sequential dependency)
vs
Payment + Inventory + Shipping (parallel, no dependency)

Use parallel when steps don't depend on each other.
```

### Caching
Mark expensive, deterministic steps as memoizable:
```json
{
  "id": "exchange_rate_lookup",
  "memoizable": true,
  "type": "sync"
}
```

## Common Anti-Patterns to Avoid

| Anti-Pattern | Problem | Solution |
|--------------|---------|----------|
| Too many goals | Hard to understand what flow is for | Keep 1-3 focused goals |
| Circular dependencies | Impossible to execute | Use DAG structure only |
| Overly granular steps | Excessive overhead | Combine related logic |
| Missing error handling | Silent failures | Explicitly mark goal vs optional steps |
| All-or-nothing parallelism | Bottlenecks on slow item | Use parallelism config to rate-limit |
| Complex predicates | Hard to debug | Keep predicates simple, use steps for logic |
