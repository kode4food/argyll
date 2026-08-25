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
| `payment-processor` | Async completion through a webhook |
| `stock-reservation` | `//argyll:wrap`, shared simulation state, and `//argyll:compensate` |
| `notification-sender` | `//argyll:step` sink over several upstream results |
| `simple-step` | Inline Lua Steps |

## Run the Complete Flow

```bash
curl -X POST http://localhost:8080/engine/flows \
  -H "Content-Type: application/json" \
  -d '{
    "id": "complete-order-flow",
    "goals": ["notification-sender"],
    "init": {
      "user_id": ["user-123"],
      "product_id": ["prod-laptop"],
      "quantity": [1]
    }
  }'
```

The Goal pulls in customer resolution, inventory lookup, order creation, stock reservation, and async payment processing through their Attribute dependencies. Payment completion takes approximately 5–15 seconds.

Inspect the Flow through the UI at `http://localhost:3001` or the API:

```bash
curl http://localhost:8080/engine/flows/complete-order-flow
```

Each service can also run directly from its directory with `go run .`. See the [documentation](https://www.argyll.app/docs/) for focused examples of Flow design, async callbacks, Work Items, and compensation.
