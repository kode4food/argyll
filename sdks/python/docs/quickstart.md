# Argyll Python SDK Quickstart

This guide will help you get started building steps and flows with the Argyll Python SDK.

## Installation

```bash
pip install argyll-sdk
```

## Your First Step

Create a simple synchronous step that processes a greeting:

```python
from argyll import Client, StepContext, AttributeType, StepResult

# Connect to the Argyll engine
client = Client("http://localhost:8080")

# Define your step handler
def handle_greeting(ctx: StepContext, args: dict) -> StepResult:
    name = args.get("name", "World")
    greeting = f"Hello, {name}!"
    return StepResult(success=True, outputs={"greeting": greeting})

# Register and start the step server
client.new_step("Greeting") \
    .required("name", AttributeType.STRING) \
    .output("greeting", AttributeType.STRING) \
    .start(handle_greeting)
```

The step server automatically:
- Registers with the engine
- Creates health check endpoint
- Handles step execution requests
- Manages error recovery

## Async Steps

For long-running operations, use async steps:

```python
from argyll import Client, StepContext, AsyncContext, AttributeType, StepResult
import threading
import time

client = Client()

def handle_async_task(ctx: StepContext, args: dict) -> StepResult:
    webhook_url = ctx.metadata.get("webhook_url")
    if not webhook_url:
        return StepResult(success=False, error="No webhook URL")

    async_ctx = AsyncContext(context=ctx, webhook_url=webhook_url)

    def background_work():
        time.sleep(5)  # Simulate long-running work
        async_ctx.success({"result": "done"})

    threading.Thread(target=background_work, daemon=True).start()
    return StepResult(success=True, outputs={"status": "started"})

client.new_step("AsyncTask") \
    .with_async_execution() \
    .output("status", AttributeType.STRING) \
    .start(handle_async_task)
```

## Script Steps

Execute Ale or Lua scripts directly:

```python
from argyll import Client, AttributeType

client = Client()

client.new_step("Double") \
    .required("value", AttributeType.NUMBER) \
    .output("result", AttributeType.NUMBER) \
    .with_script("(* value 2)") \
    .register()
```

## Running Flows

Execute a flow with multiple steps:

```python
from argyll import Client

client = Client()

# Start a flow that uses the greeting step
client.new_flow("greeting-flow-123") \
    .with_goals("greeting") \
    .with_initial_state({"name": "Alice"}) \
    .start()
```

## Builder Pattern

All builders are immutable - each method returns a new instance:

```python
builder1 = client.new_step("Test")
builder2 = builder1.with_id("custom-id")

# builder1 is unchanged
assert builder1._id == "test"
assert builder2._id == "custom-id"
```

## Advanced Features

### Conditional Execution

Use predicates to control when steps run:

```python
from argyll import ScriptLanguage

client.new_step("ConditionalStep") \
    .required("value", AttributeType.NUMBER) \
    .with_predicate(ScriptLanguage.ALE, "(> value 10)") \
    .with_endpoint("http://localhost:8081/step") \
    .register()
```

### Array Processing

Process array elements individually:

```python
client.new_step("ProcessItems") \
    .required("items", AttributeType.ARRAY) \
    .with_for_each("items") \
    .output("processed", AttributeType.STRING) \
    .with_endpoint("http://localhost:8081/process") \
    .register()
```

### Result Memoization

Cache step results for efficiency:

```python
client.new_step("ExpensiveComputation") \
    .required("input", AttributeType.NUMBER) \
    .output("result", AttributeType.NUMBER) \
    .with_memoizable() \
    .with_endpoint("http://localhost:8081/compute") \
    .register()
```

### Labels

Add metadata to steps:

```python
client.new_step("DataProcessor") \
    .with_labels({"team": "data", "env": "prod"}) \
    .with_endpoint("http://localhost:8081/process") \
    .register()
```

## Environment Variables

Configure step server settings:

```bash
export STEP_PORT=8081        # Server port (default: 8081)
export STEP_HOSTNAME=localhost  # Server hostname (default: localhost)
```

## Error Handling

All errors inherit from `ArgyllError`:

```python
from argyll import ClientError, FlowError, StepValidationError

try:
    client.new_flow("test").start()
except FlowError as e:
    print(f"Flow execution failed: {e}")
```

Custom HTTP status codes:

```python
from argyll import HTTPError

def handle_step(ctx: StepContext, args: dict) -> StepResult:
    if not args.get("token"):
        raise HTTPError(401, "Unauthorized")
    # ... process step
```

## Next Steps

- See `examples/` directory for complete working examples
- Read API reference for detailed documentation
- Check the main Argyll docs for flow orchestration concepts
