# Predicates

A **predicate** is an optional script that decides whether a step should execute given its inputs. Predicates are the supported way to model conditional execution in Argyll; imperative branching is not supported.

## Basic Concept

A predicate script evaluates to true/false:

- **True**: Step executes normally
- **False**: Step is skipped (marked as skipped, no outputs produced)

**Example:**

```json
{
  "id": "send-notification",
  "name": "Send Notification",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/notifications/send", "timeout": 5000 },
  "attributes": {
    "amount": { "role": "required", "type": "number" },
    "notification_sent": { "role": "output", "type": "boolean" }
  },
  "predicate": {
    "language": "ale",
    "script": "(> amount 100)"
  }
}
```

**Execution:**
```
Flow input: { "amount": 50 }
Predicate evaluates: (> 50 100) → false
Step is skipped, no outputs produced, no HTTP call made

Flow input: { "amount": 150 }
Predicate evaluates: (> 150 100) → true
Step executes, HTTP call made, outputs produced
```

## When Predicate Evaluates

Predicates are evaluated when work is about to start:

```
If step has for_each:
  1. Predicate is checked before initial scheduling
  2. Predicate is checked again when pending/retry work items are started
  3. If false, that work does not start
```

Example:

```json
{
  "id": "process-items",
  "name": "Process Items",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/items/process", "timeout": 5000 },
  "predicate": {
    "language": "ale",
    "script": "(> (length items) 0)"
  },
  "attributes": {
    "items": { "role": "required", "type": "array", "for_each": true },
    "processed_count": { "role": "output", "type": "number" }
  }
}
```

If `items` is empty, predicate is false and no work items run. If non-empty, work items are created and execute.

## Languages

Predicates support Ale, Lua, and JSONPath.

### Ale

Simple, purely functional language. Ideal for predicates.

```javascript
// Simple comparisons
(> amount 100)
(eq status "active")

// Logical operators
(and (> amount 100) (eq status "active"))
(or (eq region "US") (eq region "EU"))

// List operations
(> (length items) 0)
(eq (first statuses) "paid")
```

### Lua

More expressive, partial sandboxing (no I/O, os, debug).

```lua
-- Conditional expression
if amount > 100 and status == "active" then
  return true
else
  return false
end

-- More complex logic
return #items > 0 and items[1].status == "approved"
```

### JSONPath

Declarative JSON path/filter expressions. Predicate is true when the query matches at least one value (including explicit `null` matches).

```text
$.customer.active
$.items[?(@.status=="ready")]
```

## Use Cases

### Conditional Execution Based on Input

**Send expensive notification only for large orders:**

```json
{
  "id": "send-priority-notification",
  "name": "Send Priority Notification",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/notifications/priority", "timeout": 5000 },
  "attributes": {
    "amount": { "role": "required", "type": "number" },
    "notification_sent": { "role": "output", "type": "boolean" }
  },
  "predicate": {
    "language": "ale",
    "script": "(>= amount 1000)"
  }
}
```

### Skip Processing If No Work

**Only process items if there are any:**

```json
{
  "id": "batch-processor",
  "name": "Batch Processor",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/batch/process", "timeout": 5000 },
  "predicate": {
    "language": "ale",
    "script": "(> (length items) 0)"
  },
  "attributes": {
    "items": { "role": "required", "type": "array", "for_each": true },
    "batch_result": { "role": "output", "type": "string" }
  }
}
```

### Validate Input State

**Only process if customer is eligible:**

```json
{
  "id": "apply-loyalty-discount",
  "name": "Apply Loyalty Discount",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/discounts/loyalty", "timeout": 5000 },
  "attributes": {
    "years_member": { "role": "required", "type": "number" },
    "account_status": { "role": "required", "type": "string" },
    "discount_applied": { "role": "output", "type": "boolean" }
  },
  "predicate": {
    "language": "ale",
    "script": "(and (>= years_member 2) (eq account_status \"active\"))"
  }
}
```

### Complex Business Logic

```json
{
  "id": "charge-extra-fee",
  "name": "Charge Extra Fee",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/fees/charge", "timeout": 5000 },
  "attributes": {
    "order": { "role": "required", "type": "object" },
    "fee_charged": { "role": "output", "type": "boolean" }
  },
  "predicate": {
    "language": "lua",
    "script": "return order.total > 500 and order.region == 'international' and order.method == 'card'"
  }
}
```

## Handling Errors

If a predicate script fails (syntax error, runtime error), the step **fails**.

```
Predicate script: (/ 1 0)  -- Division by zero
Result: Error → Step fails → Flow fails (if goal step)
```

Write predicates carefully. Use simple logic:

```javascript
// Good: simple and clear
(> amount 100)

// Avoid: complex logic that can error
(first expensive_lookup)  -- If expensive_lookup is null
```

## Interaction with Execution Plan

The execution plan includes steps with predicates that might be skipped.

```
Execution plan: [step-a, step-b (with predicate), step-c]

At runtime:
  step-a executes
  step-b predicate evaluates
  → if true: step-b executes
  → if false: step-b is skipped
  step-c executes (regardless of step-b predicate result)
```

A skipped step produces no outputs, but downstream steps still execute (if they have other inputs available).

## Outputs When Skipped

When a predicate is false and the step is skipped:

- **No outputs produced**: Downstream steps cannot access this step's outputs
- **No HTTP call made** (sync) or no webhook expected (async)
- **No side effects**: The work isn't done

Downstream steps that depend on this step's outputs will **fail** if they need them.

**Design tip:** If downstream steps require a skipped step's outputs, provide defaults in those downstream steps or restructure your flow.

## Example Flow

```
Order processing with conditional fee:

  fetch_order
    ↓
  calculate_subtotal
    ↓
  apply_discount [predicate: apply_loyalty_discount]
    ↓
  charge_fee [predicate: (>= subtotal 1000)]
    ↓
  finalize_order
```

**Execution 1 (order subtotal: $500)**
- fetch_order: executes
- calculate_subtotal: executes
- apply_discount: predicate is false, skipped
- charge_fee: predicate is false, skipped
- finalize_order: executes

**Execution 2 (order subtotal: $1500)**
- fetch_order: executes
- calculate_subtotal: executes
- apply_discount: predicate is true, executes
- charge_fee: predicate is true, executes
- finalize_order: executes

## Predicate vs For Each Expansion

Don't confuse these:

| Feature | Predicate | For Each |
|---------|-----------|----------|
| **Purpose** | Decide if step runs | Expand into multiple work items |
| **Input** | Step attributes | Array attribute |
| **Output** | All-or-nothing (run or skip) | One output per item, aggregated |
| **Use for** | Conditional logic | Parallelism and batching |

A step can have both:

```json
{
  "id": "process-items-if-non-empty",
  "name": "Process Items If Non Empty",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/items/process", "timeout": 5000 },
  "predicate": {
    "language": "ale",
    "script": "(> (length items) 0)"
  },
  "attributes": {
    "items": { "role": "required", "type": "array", "for_each": true },
    "processed_count": { "role": "output", "type": "number" }
  }
}
```

Predicate evaluates first (is there anything to process?), then for_each expansion happens (create work items).

## Common Patterns

### Tier-Based Processing

```json
{
  "id": "process-premium-customers",
  "name": "Process Premium Customers",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/customers/premium/process", "timeout": 5000 },
  "attributes": {
    "customer_tier": { "role": "required", "type": "string" },
    "processed": { "role": "output", "type": "boolean" }
  },
  "predicate": {
    "language": "ale",
    "script": "(eq customer_tier \"premium\")"
  }
}
```

### Time-Sensitive Gating

```json
{
  "id": "send-time-sensitive-offer",
  "name": "Send Time Sensitive Offer",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/offers/send", "timeout": 5000 },
  "attributes": {
    "current_epoch": { "role": "required", "type": "number" },
    "offer_expiry": { "role": "required", "type": "number" },
    "sent": { "role": "output", "type": "boolean" }
  },
  "predicate": {
    "language": "lua",
    "script": "return current_epoch < offer_expiry"
  }
}
```

### Feature Flags

```json
{
  "id": "experimental-feature",
  "name": "Experimental Feature",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/features/experimental", "timeout": 5000 },
  "attributes": {
    "feature_enabled": { "role": "required", "type": "boolean" },
    "executed": { "role": "output", "type": "boolean" }
  },
  "predicate": {
    "language": "ale",
    "script": "(eq feature_enabled true)"
  }
}
```

### Data Validation

```json
{
  "id": "process-if-valid",
  "name": "Process If Valid",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/validation/process", "timeout": 5000 },
  "attributes": {
    "email": { "role": "required", "type": "string" },
    "processed": { "role": "output", "type": "boolean" }
  },
  "predicate": {
    "language": "lua",
    "script": "return email:find('@') ~= nil"
  }
}
```

## Troubleshooting

**Q: Predicate is false but downstream step still fails?**
A: Downstream step depends on outputs from the skipped step. Either:
- Provide default values in the downstream step's input
- Restructure flow so downstream step doesn't need skipped step's outputs

**Q: Why does the step show as "skipped" in the UI?**
A: The predicate evaluated to false. Check the predicate logic and flow inputs.

**Q: Can I set a timeout for predicate evaluation?**
A: No, predicates are simple scripts. Keep them simple to avoid timeouts.

## Best Practices

1. **Keep predicates simple**: Use basic comparisons and logical operators
2. **Avoid external calls**: Predicates run synchronously; don't do I/O or HTTP
3. **Test predicates**: Verify edge cases (empty arrays, null values, etc.)
4. **Document intent**: Add comments explaining the business logic
5. **Handle missing inputs**: Predicates should account for optional inputs that might be absent
