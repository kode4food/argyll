# Flow Design Patterns

This guide covers practical patterns for designing flows effectively.

## Start with Goals

Define clear goal steps - these are the outcomes you actually need. The engine builds the minimal execution plan to reach them.

**Why:** Avoids speculative work and keeps flows focused.

**Example:**
```
Goal: process_payment + send_confirmation
Dependencies: validate_customer, lookup_inventory

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
    "notify_user": { "role": "optional", "type": "boolean", "input": { "default": "true" } },
    "timeout_seconds": { "role": "optional", "type": "number", "input": { "default": "30" } }
  }
}
```

**Why:** Enables the same step to work in multiple flows without duplication.

## Input Collection

Inputs can choose how to collect matching upstream outputs with `input.collect`. If omitted, `first` is used, which preserves the existing behavior.

| Collect | Starts when | Runtime value |
| ------- | ----------- | ------------- |
| `first` | The first provider value is available | The first value |
| `last` | All potential providers are terminal and at least one value exists | The last produced value |
| `some` | All potential providers are terminal and at least one value exists | An array of produced values |
| `all` | All potential providers complete successfully with values | An array of produced values |
| `none` | All potential providers are terminal and no value exists | No value, or the optional default if set |

`first` and `last` are singleton modes. `some` and `all` are list modes. For `input.for_each`, singleton modes treat the selected value as the array to iterate, while list modes iterate the collected array itself.

Collection policies also affect lazy execution. Once a consumer no longer needs more values, providers that only fed that consumer can be skipped. This happens naturally for `first` after the first value, for failed `all` collections that can no longer succeed, and for optional inputs after their timeout window closes.

```json
{
  "attributes": {
    "quotes": { "role": "required", "type": "array", "input": { "collect": "some", "for_each": true } },
    "best_quote": { "role": "output", "type": "object" }
  }
}
```

Initial flow values use an array per attribute name. This makes a scalar array value unambiguous:

```json
{
  "init": {
    "customer_id": ["cust-123"],
    "quotes": [[{ "id": "quote-1" }, { "id": "quote-2" }]]
  }
}
```

### Optional Timeouts

Optional inputs can define `input.timeout` in milliseconds. The timeout starts when the step's required inputs are ready. If the optional input is still not resolved when the timeout expires, the step closes that input's collection window and starts with the best value available for its collection policy.

```json
{
  "attributes": {
    "profile": { "role": "required", "type": "object" },
    "preferences": {
      "role": "optional",
      "type": "object",
      "input": {
        "default": "{}",
        "timeout": 2000
      }
    },
    "rendered_email": { "role": "output", "type": "string" }
  }
}
```

| Collect | At timeout |
| ------- | ---------- |
| `first` | Uses the first value if one arrived; otherwise uses the default or omits the input |
| `last` | Uses the latest value that arrived; otherwise uses the default or omits the input |
| `some` | Uses the values that arrived if at least one exists; otherwise uses the default or omits the input |
| `all` | Uses the collected values only if every provider completed successfully; otherwise uses the default or omits the input |
| `none` | Uses the default or omits the input if no value arrived |

The timeout decision is local to that step execution. Later values can still enter the flow and satisfy other consumers, but this step will not restart to use them.

## For Each and Parallelism

Use `for_each` on array inputs to process multiple items in parallel. Outputs are aggregated.

```json
{
  "attributes": {
    "order_items": { "role": "required", "type": "array", "input": { "for_each": true } },
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

**Why:** Encapsulation and reuse. The sub-flow has its own execution plan and goals, and that child plan is fixed when the parent plan is compiled.

**Pattern:** Create sub-flows for domain-specific logic (authentication, payment, validation) that might be reused across multiple parent flows.

## Idempotency

Idempotency for async completion is handled by the engine.

Use the engine-provided metadata and webhook token path:
- `flow_id`
- `step_id`
- `receipt_token` (also encoded in `/webhook/{flow_id}/{step_id}/{token}`)

The engine rejects duplicate completions for the same token, so normal step retries can use the standard webhook/token path without extra duplicate-detection logic.

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
