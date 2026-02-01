# Payment Processor (Python)

Async payment processor step example using the Argyll Python SDK.

## Features

- **Async execution** - Processes payments in background
- **Realistic simulation** - Random processing time (5-10 seconds)
- **Random outcomes** - 50% success, 50% failure
- **Rich metadata** - Returns payment ID, timestamps, status

## Inputs

- `order` (object) - Order details with:
  - `id` - Order ID
  - `grand_total` - Payment amount
  - `payment_method` - Payment method used

## Outputs

- `payment_result` (object) - Payment result with:
  - `order_id` - Original order ID
  - `amount` - Payment amount
  - `status` - Payment status (completed)
  - `payment_id` - Generated payment ID
  - `processed_at` - ISO timestamp

## Running

### With Docker Compose

```bash
docker compose up payment-processor
```

### Standalone

```bash
cd examples/payment-processor
pip install -r requirements.txt
python main.py
```

## Environment Variables

- `ARGYLL_ENGINE_URL` - Engine URL (default: http://localhost:8080)
- `STEP_PORT` - Server port (default: 8085)
- `STEP_HOSTNAME` - Server hostname (default: payment-processor)

## Example Usage

The payment processor is designed to work in a flow with order creation:

```python
from argyll import Client

client = Client("http://localhost:8080")

# Start a flow that processes an order and payment
client.new_flow("order-flow-123") \
    .with_goals("payment-processor") \
    .with_initial_state({
        "order": {
            "id": "ORD-123",
            "grand_total": 99.99,
            "payment_method": "credit_card"
        }
    }) \
    .start()
```

## Implementation Notes

- Uses Python's `threading` module for background processing
- Simulates payment gateway delays with `time.sleep()`
- Returns immediately from handler, completes via webhook
- Demonstrates proper async context usage
