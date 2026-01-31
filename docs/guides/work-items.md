# Work Items

When a step has a `for_each` attribute marked as an array, it expands into multiple **work items**. Each work item executes independently and produces its own outputs. This guide explains how it works and how to configure parallelism.

## For Each Expansion

### Basic Concept

When a step input is marked `for_each` and provided as an array, the engine creates one work item per array element.

**Step definition:**
```json
{
  "id": "process-item",
  "type": "sync",
  "http": { "endpoint": "..." },
  "attributes": {
    "items": {
      "required": true,
      "for_each": true
    },
    "processed": {
      "required": false
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
    "regions": { "for_each": true },
    "products": { "for_each": true }
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
  "type": "sync",
  "work_config": {
    "parallelism": 5
  },
  "attributes": {
    "items": { "required": true, "for_each": true }
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
    "backoff_ms": 100,
    "max_backoff_ms": 5000,
    "backoff_type": "exponential"
  }
}
```

| Field | Meaning |
|-------|---------|
| `parallelism` | Max concurrent work items (default: 1) |
| `max_retries` | Retries per work item on failure |
| `backoff_ms` | Initial backoff (milliseconds) |
| `max_backoff_ms` | Maximum backoff |
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

If some work items fail, the step status reflects the aggregated result.

```
5 work items
4 succeed
1 fails

Step status: "partial_failure" (or "failed" depending on your flow)
Outputs aggregated from successful items
Failed item recorded in error logs
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

- Predicate evaluates **once** before work items are created
- If predicate is false, entire step is skipped (no work items run)
- If predicate is true, all work items execute

```json
{
  "id": "batch-process",
  "predicate": {
    "language": "ale",
    "script": "(> (length items) 0)"
  },
  "attributes": {
    "items": { "for_each": true }
  }
}
```

This step only runs if the items array is non-empty. If empty, the step is skipped entirely.

## API Details

When a step handler receives a work item:

```json
{
  "step_id": "process-item",
  "arguments": {
    "item": { "id": "item-1", "value": 10 }
  },
  "metadata": {
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

This counts as a handled failure and triggers retries (if configured).

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
A: After the configured timeout, it's marked failed. The step continues with remaining items if you have retries configured.
