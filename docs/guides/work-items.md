# Work Items

When a step has a `for_each` attribute marked as an array, it expands into multiple **work items**. Each work item executes independently and produces its own outputs. This guide explains how it works and how to configure parallelism.

## For Each Expansion

### Basic Concept

When a step input is marked `for_each` and provided as an array, the engine creates one work item per array element.

**Step definition:**
```json
{
  "id": "process-item",
  "name": "Process Item",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/items/process", "timeout": 5000 },
  "attributes": {
    "items": {
      "role": "required",
      "type": "array",
      "for_each": true
    },
    "processed": {
      "role": "output",
      "type": "number"
    }
  }
}
```

**Flow input:**
```json
{
  "items": [
    { "id": "item-1", "value": 10 },
    { "id": "item-2", "value": 20 },
    { "id": "item-3", "value": 15 }
  ]
}
```

**Engine creates:**
- Work item 1: `{ "items": { "id": "item-1", "value": 10 } }`
- Work item 2: `{ "items": { "id": "item-2", "value": 20 } }`
- Work item 3: `{ "items": { "id": "item-3", "value": 15 } }`

Each work item executes independently, and the handler runs 3 times (once per item).

### Multiple For Each Attributes

A step can have multiple `for_each` attributes. The engine computes the **Cartesian product**.

**Step definition:**
```json
{
  "attributes": {
    "regions": { "role": "required", "type": "array", "for_each": true },
    "products": { "role": "required", "type": "array", "for_each": true }
  }
}
```

**Flow input:**
```json
{
  "regions": ["US", "EU", "APAC"],
  "products": ["A", "B"]
}
```

**Work items created:** 3 × 2 = 6
```
{ "regions": "US", "products": "A" }
{ "regions": "US", "products": "B" }
{ "regions": "EU", "products": "A" }
{ "regions": "EU", "products": "B" }
{ "regions": "APAC", "products": "A" }
{ "regions": "APAC", "products": "B" }
```

## Work Tokens

Each work item has a unique **token**. The token is used to:

- Track which work item is executing
- Associate completion events with the correct work item
- Merge outputs when the step completes

**Flow of a work item:**

```
1. work_item created with token = "work-abc-123"
2. Engine calls step handler with:
   {
     "arguments": { "region": "US", "product": "A" },
     "metadata": {
       "flow_id": "flow-123",
       "step_id": "process-item",
       "receipt_token": "work-abc-123"
     }
   }
3. Handler processes and returns outputs
4. work_succeeded event recorded with token = "work-abc-123"
5. Engine associates outputs with work item 1
6. When all work items complete, outputs are merged
```

### Tokens and Retries

Each work item has a token that the engine uses to track completion. If a work item is retried, the engine handles token management automatically.

**You don't need to worry about this.** The engine ensures that:
- Duplicate webhook calls with the same token are rejected
- Retries are safe and won't cause duplicate work
- Late responses from old attempts are handled correctly

See [Memoization](memoization.md) for when to mark steps as deterministic (results never change for same inputs).

## Output Aggregation

When a step with `for_each` completes, its outputs are **aggregated** back into flow attributes.

### Single Output Attribute

If the step produces a single output:

```
Work item 1: processed = 100
Work item 2: processed = 200
Work item 3: processed = 150

Aggregated output (in flow attributes):
"processed": [
  { "items": { "id": "item-1", ... }, "processed": 100 },
  { "items": { "id": "item-2", ... }, "processed": 200 },
  { "items": { "id": "item-3", ... }, "processed": 150 }
]
```

Each element in the aggregated array includes:
- The `for_each` input values (for reference)
- The step's output

### Multiple Output Attributes

If the step produces multiple outputs, each is aggregated separately:

```
Work item 1: result = "ok", processed_at = "2025-01-30T10:00Z"
Work item 2: result = "ok", processed_at = "2025-01-30T10:01Z"

Aggregated outputs:
"result": [
  { "items": {...}, "result": "ok" },
  { "items": {...}, "result": "ok" }
],
"processed_at": [
  { "items": {...}, "processed_at": "2025-01-30T10:00Z" },
  { "items": {...}, "processed_at": "2025-01-30T10:01Z" }
]
```

## Parallelism Control

By default, work items are processed sequentially. Use **WorkConfig** to control concurrency.

### Step Definition with Parallelism

```json
{
  "id": "process-items",
  "name": "Process Items",
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/items/process", "timeout": 5000 },
  "work_config": {
    "parallelism": 5
  },
  "attributes": {
    "items": { "role": "required", "type": "array", "for_each": true },
    "processed": { "role": "output", "type": "number" }
  }
}
```

**Result:** Up to 5 work items process concurrently.

### Parallelism Options

```json
{
  "work_config": {
    "parallelism": 10,
    "max_retries": 3,
    "backoff": 100,
    "max_backoff": 5000,
    "backoff_type": "exponential"
  }
}
```

| Field | Meaning |
|-------|---------|
| `parallelism` | Max concurrent work items (`<= 0` or omitted defaults to `1`) |
| `max_retries` | Retries per work item on failure |
| `backoff` | Initial backoff (milliseconds) |
| `max_backoff` | Maximum backoff (milliseconds) |
| `backoff_type` | `fixed`, `linear`, or `exponential` |

### When to Increase Parallelism

- **I/O-bound steps**: Increase to 10-50 to avoid blocking on network
- **CPU-bound steps**: Keep low (1-4) to avoid overload
- **Downstream capacity**: Match your downstream system's rate limit

**Example:**
```
Inventory API supports 10 concurrent requests
→ parallelism: 10
```

## Partial Failure

If any work item fails permanently, the step ends in `failed` once all work
items reach terminal states.

```
5 work items
4 succeed
1 fails

Step status: "failed"
Successful work outputs remain recorded on their work items
Failure reason is stored on the step execution
```

## Use Cases

### Batch Processing

```
items = [order1, order2, ..., order100]
→ process-order [for_each]
→ confirm-shipment (operates on aggregated processed items)
```

### Parallel API Calls

```
customer_ids = ["cust-1", "cust-2", "cust-3", ...]
→ lookup-customer [for_each, parallelism: 20]
→ create-notification (once all lookups complete)
```

### Fan-Out / Fan-In

```
inventory_items = [item1, item2, item3, ...]
→ reserve-inventory [for_each, parallelism: 5]
→ confirm-reservations (aggregate results)
→ send-order-confirmation
```

## Interaction with Predicates

If a step has both a `predicate` and `for_each`:

- Predicate is evaluated before initial work-item scheduling
- Predicate is also checked when pending/retry work items are about to start
- If predicate is false, affected work does not start (or is skipped at step
  level before any work items are started)

```json
{
  "id": "batch-process",
  "name": "Batch Process",
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

This step only runs if the items array is non-empty. If empty, the step is skipped entirely.

## API Details

When a step handler receives a work item:

```json
{
  "arguments": {
    "item": { "id": "item-1", "value": 10 }
  },
  "metadata": {
    "step_id": "process-item",
    "flow_id": "flow-123",
    "receipt_token": "work-abc-123"
  }
}
```

**Important:** Always return a 200 OK response (sync steps) or accept the request (async steps). The handler must return successfully. If you want to mark a work item as failed, return:

```json
{
  "success": false,
  "error": "Item validation failed"
}
```

This records a handled, permanent failure for that work item.

## Common Patterns

### Process and Aggregate

```
items[for_each]
├─ validate-item
├─ transform-item
├─ aggregate-results (operates on transformed items)
└─ send-report
```

### Bulk Operations with Rate Limiting

```
users[for_each, parallelism: 10]
├─ notify-user
├─ log-notification
└─ finalize (once all notifications sent)
```

### Conditional Bulk Processing

```
orders[for_each, parallelism: 5]
├─ predicate: order.total > 100
├─ charge-premium-fee
└─ record-fees (aggregated results)
```

## Troubleshooting

**Q: Why are my work items executing sequentially even though I set parallelism?**
A: Check that the downstream system isn't bottlenecked. Parallelism is limited by both the config AND the system's capacity.

**Q: How do I know which aggregated output corresponds to which input?**
A: Each aggregated output includes the `for_each` input values. In the code above, each element has both `items` (the input) and `processed` (the output).

**Q: Can I modify the parallelism at runtime?**
A: No, it's set in the step definition. Create a new step version if you need to change it.

**Q: What happens if a work item never completes?**
A: Completion behavior is controlled by retry settings and result reporting.
Transient failures (`work_not_completed`) retry until retry budget is exhausted;
permanent failures (`work_failed`) fail that work item immediately.
