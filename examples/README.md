# Argyll Example Steps

This directory contains example step implementations demonstrating different step types and flow patterns.

## Quick Start

```bash
# Start the entire system with all examples
docker compose up

# Or start just the engine and specific examples
docker compose up valkey argyll-engine user-resolver inventory-resolver
```

## Example Steps Overview

### HTTP-Based Steps (Sync)

#### 1. **user-resolver** (Resolver Pattern)
Looks up user information from an in-memory database.

**Type**: Sync HTTP, Resolver (no required inputs)

**Optional Inputs**:
- `user_id` (string) - User ID to look up (defaults to "user-123")

**Outputs**:
- `user_info` (object) - User details including account type, credit limit, preferred notification method

**Available Users**:
- `user-123` - Alice Johnson (premium, $5000 credit limit, email preferred)
- `user-456` - Bob Smith (standard, $1000 credit limit, SMS preferred)
- `user-789` - Carol Williams (premium, $10000 credit limit, webhook preferred)

**Example Flow**:
```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "user-lookup-test",
    "goals": ["user-resolver"],
    "init": {"user_id": "user-456"}
  }'
```

---

#### 2. **inventory-resolver** (Resolver Pattern)
Retrieves product information and stock levels.

**Type**: Sync HTTP, Resolver

**Optional Inputs**:
- `product_id` (string) - Product ID to look up (defaults to "prod-laptop")

**Outputs**:
- `product_info` (object) - Product details including price, stock levels, order limits

**Available Products**:
- `prod-laptop` - Professional Laptop ($1299.99, 50 in stock)
- `prod-mouse` - Wireless Mouse ($29.99, 200 in stock)
- `prod-keyboard` - Mechanical Keyboard ($149.99, 75 in stock)
- `prod-monitor` - 4K Monitor 27" ($449.99, 30 in stock)
- `prod-headphones` - Noise-Canceling Headphones ($249.99, **OUT OF STOCK**)

**Example Flow**:
```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "inventory-check",
    "goals": ["inventory-resolver"],
    "init": {"product_id": "prod-headphones"}
  }'
```

**Expected Result**: Flow will complete with out-of-stock product info.

---

#### 3. **order-creator** (Processor Pattern)
Creates an order with comprehensive business logic validation.

**Type**: Sync HTTP, Processor

**Required Inputs**:
- `user_info` (object) - User details from user-resolver
- `product_info` (object) - Product details from inventory-resolver

**Optional Inputs**:
- `quantity` (number) - Order quantity (defaults to 1)
- Optional inputs can also define a timeout (milliseconds) in the step definition for default fallback behavior
- Timeout starts when the first upstream provider step for that attribute begins work
- If the timeout expires, the step can continue using its optional default value
- The default is step-local only (it does not become a flow attribute)

**Outputs**:
- `order` (object) - Order details including pricing breakdown

**Business Logic**:
- Validates stock availability
- Enforces min/max order quantities
- Calculates shipping based on weight
- Applies 8% tax
- Checks credit limits for standard accounts

**Example Flow (Success)**:
```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "create-order-success",
    "goals": ["order-creator"],
    "init": {
      "user_id": "user-123",
      "product_id": "prod-mouse",
      "quantity": 2
    }
  }'
```

**Example Flow (Credit Limit Exceeded)**:
```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "create-order-credit-fail",
    "goals": ["order-creator"],
    "init": {
      "user_id": "user-456",
      "product_id": "prod-laptop",
      "quantity": 2
    }
  }'
```

**Expected Result**: Flow will fail - Bob (standard account, $1000 limit) cannot afford 2 laptops ($2599.98 + tax + shipping).

**Example Flow (Out of Stock)**:
```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "create-order-oos",
    "goals": ["order-creator"],
    "init": {
      "user_id": "user-123",
      "product_id": "prod-headphones"
    }
  }'
```

**Expected Result**: Flow will fail - headphones are out of stock.

---

#### 4. **stock-reservation** (Processor with Shared State)
Reserves inventory with thread-safe stock tracking.

**Type**: Sync HTTP, Processor

**Required Inputs**:
- `order` (object) - Order details from order-creator

**Outputs**:
- `reservation` (object) - Reservation details with 30-minute expiration

**Important**: This step maintains shared state across invocations. Stock levels decrease with each reservation.

**Example Flow**:
```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "reserve-stock",
    "goals": ["stock-reservation"],
    "init": {
      "user_id": "user-123",
      "product_id": "prod-mouse",
      "quantity": 3
    }
  }'
```

**Testing Overselling**:
Run the same flow multiple times to deplete stock and trigger insufficient stock errors.

---

#### 5. **notification-sender** (Collector Pattern)
Sends notifications via user's preferred method.

**Type**: Sync HTTP, Collector

**Required Inputs**:
- `payment_result` (object) - Payment confirmation
- `reservation` (object) - Stock reservation
- `user_info` (object) - User details (for notification preferences)

**Outputs**:
- `notification_result` (object) - Notification delivery confirmation

**Notification Channels**:
- Email (Alice's preference)
- SMS (Bob's preference)
- Webhook (Carol's preference)
- Always sends backup email for audit trail

**Example Flow**:
```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "notify-user",
    "goals": ["notification-sender"],
    "init": {
      "user_id": "user-456",
      "product_id": "prod-keyboard"
    }
  }'
```

**Expected Result**: SMS notification to Bob + backup email.

---

### HTTP-Based Steps (Async)

#### 6. **payment-processor** (Async Processor)
Processes payment asynchronously with webhook callback.

**Type**: Async HTTP, Processor

**Required Inputs**:
- `order` (object) - Order to process payment for

**Outputs**:
- `payment_result` (object) - Payment confirmation or error

**Behavior**:
- Returns immediately (async)
- Processes for 5-15 seconds
- 90% success rate, 10% failure rate
- Fails with realistic reasons: insufficient funds, card declined, fraud detection, etc.

**Example Flow**:
```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "process-payment",
    "goals": ["payment-processor"],
    "init": {
      "user_id": "user-123",
      "product_id": "prod-laptop"
    }
  }'
```

**Watch the logs**: Payment will complete asynchronously after 5-15 seconds. Run multiple times to see both success and failure cases.

---

### Script-Based Steps

#### 7. **simple-step** (Script Examples)
Registers three script-based steps demonstrating Ale and Lua.

**Type**: Script (no HTTP service required)

**Included Steps**:

##### a. **text-formatter** (Ale)
Formats text with user name prefix.

**Inputs**: `text` (string), `name` (string)
**Outputs**: `formatted_text` (string)

**Example**:
```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "format-text",
    "goals": ["text-formatter"],
    "init": {
      "text": "Welcome to Argyll!",
      "name": "Alice Johnson"
    }
  }'
```

**Expected Output**: `"[ALICE] Welcome to Argyll!"`

##### b. **price-calculator** (Ale)
Calculates pricing with tax and shipping.

**Inputs**: `quantity` (number), `unit_price` (number)
**Outputs**: `subtotal`, `tax`, `shipping`, `total` (all numbers)

**Example**:
```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "calc-price",
    "goals": ["price-calculator"],
    "init": {
      "quantity": 3,
      "unit_price": 29.99
    }
  }'
```

**Expected Output**: Subtotal $89.97, tax $7.20, shipping $9.99 (< 5 qty), total $107.16

##### c. **eligibility-checker** (Lua)
Checks loan eligibility based on age, income, and credit score.

**Inputs**: `age` (number), `income` (number), `credit_score` (number)
**Outputs**: `eligible` (boolean), `reason` (string), `risk_level` (string)

**Example (Eligible)**:
```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "check-eligible-pass",
    "goals": ["eligibility-checker"],
    "init": {
      "age": 35,
      "income": 75000,
      "credit_score": 720
    }
  }'
```

**Expected Output**: `eligible: true, reason: "all criteria met", risk_level: "medium"`

**Example (Ineligible - Too Young)**:
```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "check-eligible-fail",
    "goals": ["eligibility-checker"],
    "init": {
      "age": 17,
      "income": 50000,
      "credit_score": 700
    }
  }'
```

**Expected Output**: `eligible: false, reason: "age below minimum (18)"`

---

## Complete E-Commerce Flow

Demonstrates goal-driven execution with lazy evaluation:

```bash
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "complete-order-flow",
    "goals": ["notification-sender"],
    "init": {
      "user_id": "user-123",
      "product_id": "prod-laptop",
      "quantity": 1
    }
  }'
```

**Execution Plan (Lazy Evaluation)**:
1. `user-resolver` - Resolves user-123 (Alice)
2. `inventory-resolver` - Resolves prod-laptop
3. `order-creator` - Creates order (validates, calculates pricing)
4. `stock-reservation` - Reserves 1 laptop
5. `payment-processor` - Processes payment (async, 5-15 sec)
6. `notification-sender` - Sends email to Alice + backup email

**Total Time**: ~5-15 seconds (dominated by async payment)

---

## Testing Different Scenarios

### Test Payment Failure Recovery
Run the complete flow multiple times. ~10% will fail at payment processing:

```bash
for i in {1..10}; do
  curl -X POST http://localhost:8080/engine/flow \
    -H "Content-Type: application/json" \
    -d "{\"id\": \"order-$i\", \"goals\": [\"payment-processor\"], \"init\": {\"user_id\": \"user-123\", \"product_id\": \"prod-mouse\"}}"
done
```

Watch logs to see successful payments and various failure reasons.

### Test Stock Depletion
Reserve all available stock of a product:

```bash
# Monitors have 30 in stock, reserve 35 to trigger failure
for i in {1..35}; do
  curl -X POST http://localhost:8080/engine/flow \
    -H "Content-Type: application/json" \
    -d "{\"id\": \"reserve-$i\", \"goals\": [\"stock-reservation\"], \"init\": {\"user_id\": \"user-123\", \"product_id\": \"prod-monitor\", \"quantity\": 1}}"
  echo "Reservation $i"
done
```

First 30 will succeed, remaining 5 will fail with "insufficient stock".

### Test Credit Limit Enforcement
Try ordering as a standard user (Bob) with limited credit:

```bash
# Bob has $1000 credit limit
# Try ordering expensive laptop ($1299.99 + tax + shipping = ~$1410)
curl -X POST http://localhost:8080/engine/flow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "credit-fail",
    "goals": ["order-creator"],
    "init": {
      "user_id": "user-456",
      "product_id": "prod-laptop"
    }
  }'
```

Flow will fail at order-creator with "order total exceeds credit limit".

---

## Monitoring Flows

### View All Flows
```bash
curl http://localhost:8080/engine/flow
```

### View Specific Flow
```bash
curl http://localhost:8080/engine/flow/complete-order-flow
```

### WebSocket Live Updates
Open the Web UI at http://localhost:3001 to watch flows execute in real-time.

---

## Development

### Run Single Example Locally
```bash
cd examples/user-resolver
go run main.go
```

### Register Script Steps
```bash
cd examples/simple-step
go run main.go
```

This registers all three script steps (text-formatter, price-calculator, eligibility-checker) without needing separate services.

---

## Key Concepts Demonstrated

1. **Resolvers** - Produce outputs on demand (user-resolver, inventory-resolver)
2. **Processors** - Transform inputs to outputs (order-creator, stock-reservation, payment-processor)
3. **Collectors** - Consume inputs, perform side effects (notification-sender)
4. **Lazy Evaluation** - Only executes steps needed for the goal
5. **Async Flows** - Payment processor shows webhook-based async execution
6. **Error Handling** - Multiple failure scenarios (out of stock, credit limits, payment failures)
7. **Shared State** - Stock reservation maintains inventory across flows
8. **Script Steps** - Ale and Lua for lightweight logic without HTTP services
