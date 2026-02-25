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
    "customer_id": { "role": "required", "type": "string" },
    "notify_user": { "role": "optional", "type": "boolean", "default": "true" },
    "timeout_seconds": { "role": "optional", "type": "number", "default": "30" }
  }
}
```

**Why:** Enables the same step to work in multiple flows without duplication.

### Optional Timeouts (Step-Local Fallback)

Optional inputs can define a `timeout` (milliseconds) to let a step continue with its own default value when an upstream provider is slow.

```json
{
  "attributes": {
    "profile": { "role": "required", "type": "object" },
    "preferences": {
      "role": "optional",
      "type": "object",
      "default": "{}",
      "timeout": 2000
    },
    "rendered_email": { "role": "output", "type": "string" }
  }
}
```

**Behavior:**
- `timeout: 0` means there is no wait window for that optional input (use the upstream value only if it is already present when the step is ready; otherwise use the default or omit the input)
- If `timeout` is greater than 0, the step waits up to that long for the optional value
- The timeout clock starts when the step's required inputs are satisfied (or at flow start if the step has no required inputs)
- If the timeout expires first, this step can proceed with its default
- That default choice is step-local and sticky for this step execution, even if the real attribute still arrives before the step starts
- Other steps that require the real attribute still wait for it

**Why:** Improves latency for downstream steps that can tolerate a fallback while preserving correctness for strict consumers.

## For Each and Parallelism

Use `for_each` on array inputs to process multiple items in parallel. Outputs are aggregated.

```json
{
  "attributes": {
    "order_items": { "role": "required", "type": "array", "for_each": true },
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

Use predicates to conditionally skip steps based on flow state.

```json
{
  "id": "apply_vip_discount",
  "name": "Apply VIP Discount",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/discounts/apply", "timeout": 5000 },
  "attributes": {
    "order_total": { "role": "required", "type": "number" },
    "customer_tier": { "role": "required", "type": "string" },
    "discount_applied": { "role": "output", "type": "boolean" }
  },
  "predicate": {
    "language": "ale",
    "script": "(and (> order_total 1000) (eq customer_tier \"gold\"))"
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
  "name": "Authorize Payment",
  "type": "flow",
  "flow": {
    "goals": ["validate_payment_method"]
  },
  "attributes": {
    "cardnum": { "role": "required", "mapping": { "name": "card_number" } },
    "security_code": { "role": "required", "mapping": { "name": "cvv" } },
    "authorization_result": {
      "role": "output",
      "mapping": { "name": "is_valid" }
    }
  }
}
```

**Why:** Encapsulation and reuse. The sub-flow has its own execution plan and goals.

**Pattern:** Create sub-flows for domain-specific logic (authentication, payment, validation) that might be reused across multiple parent flows.

## Idempotency

Idempotency for async completion is handled by the engine.

Use the engine-provided metadata and webhook token path:
- `flow_id`
- `step_id`
- `receipt_token` (also encoded in `/webhook/{flow_id}/{step_id}/{token}`)

The engine rejects duplicate completions for the same token, so you do not need to implement your own duplicate-detection pattern for normal step retries.

See [Async Steps Guide](./async-steps.md) for details.

## Minimize Cross-Cutting Concerns

Avoid flows where many steps depend on each other in complex ways. Instead:
- **Keep dependencies linear when possible**: A → B → C
- **Use sub-flows** for clusters of related steps
- **Use predicates** for conditional execution

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
  "name": "Exchange Rate Lookup",
  "memoizable": true,
  "type": "sync",
  "http": { "endpoint": "https://api.exchangerate-api.com/rates", "timeout": 5000 },
  "attributes": {
    "base_currency": { "role": "required", "type": "string" },
    "quote_currency": { "role": "required", "type": "string" },
    "rate": { "role": "output", "type": "number" }
  }
}
```

## Common Anti-Patterns to Avoid

| Anti-Pattern | Problem | Solution |
|--------------|---------|----------|
| Too many goals | Hard to understand what flow is for | Keep 1-3 focused goals |
| Overly granular steps | Excessive overhead | Combine related logic |
| Missing error handling | Silent failures | Explicitly mark goal vs optional steps |
| All-or-nothing parallelism | Bottlenecks on slow item | Use parallelism config to rate-limit |
| Complex predicates | Hard to debug | Keep predicates simple, use steps for logic |
