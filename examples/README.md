# Argyll Examples

The examples form one runnable commerce Flow while exercising sync, async, script, generated, stateful, and compensated Steps.

## Run

```bash
docker compose up
```

| Example | Demonstrates |
|---|---|
| `user-resolver` | Sync source for customer data |
| `inventory-resolver` | Sync product and stock lookup |
| `order-creator` | Validation and derived order values |
| `payment-processor` | Async completion through a webhook and refund compensation |
| `stock-reservation` | Stock reservation and compensation |
| `notification-sender` | `//argyll:step` sink over several upstream results |
| `simple-step` | Inline Lua Steps |

Order creation, stock reservation, payment processing, and notification sending keep successful results in memory and replay them when the same work token is retried.

## Run the Complete Flow

Set `ARGYLL_URL` to your Argyll address, then start the Flow:

```bash
curl -X POST "$ARGYLL_URL/engine/flows" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "complete-order-flow",
    "goals": {"steps": ["notification-sender"]},
    "init": {
      "user_id": ["user-123"],
      "product_id": ["prod-laptop"],
      "quantity": [1]
    }
  }'
```

`goals` is a chain of fallback goal sets: every one of `steps` must succeed, and the set in `else` is attempted only when `steps` becomes impossible, reusing the work already done. The Goal pulls in customer resolution, inventory lookup, order creation, stock reservation, and async payment processing through their Attribute dependencies. Payment completion takes approximately 5–15 seconds.

Inspect the Flow through the UI at `$ARGYLL_URL` or the API:

```bash
curl "$ARGYLL_URL/engine/flows/complete-order-flow"
```

Each service can also run directly from its directory with `go run .`. See the [documentation](https://www.argyll.app/docs/) for focused examples of Flow design, async callbacks, Work Items, and compensation.
