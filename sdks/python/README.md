# Argyll Python SDK

Python SDK for building steps and flows with the Argyll distributed orchestrator.

## Installation

```bash
pip install argyll-sdk
```

## Quick Start

### Define a Sync Step

```python
from argyll import Client, StepContext, AttributeType, StepResult

client = Client("http://localhost:8080/engine")

def handle_greeting(ctx: StepContext, args: dict) -> StepResult:
    name = args.get("name", "World")
    greeting = f"Hello, {name}!"
    return StepResult(success=True, outputs={"greeting": greeting})

client.new_step("Greeting") \
    .required("name", AttributeType.STRING) \
    .output("greeting", AttributeType.STRING) \
    .start(handle_greeting)
```

### Define an Async Step

```python
from argyll import Client, StepContext, AsyncContext, AttributeType, StepResult
import threading

client = Client("http://localhost:8080/engine")

def handle_async_task(ctx: StepContext, args: dict) -> StepResult:
    # Extract webhook URL from metadata
    webhook_url = ctx.metadata.get("webhook_url")
    if not webhook_url:
        return StepResult(success=False, error="No webhook URL")

    async_ctx = AsyncContext(context=ctx, webhook_url=webhook_url)

    # Start background processing
    def process():
        try:
            # Do long-running work
            result = {"status": "completed"}
            async_ctx.success(result)
        except Exception as e:
            async_ctx.fail(str(e))

    threading.Thread(target=process).start()

    # Return immediately
    return StepResult(success=True, outputs={})

client.new_step("AsyncTask") \
    .with_async_execution() \
    .output("status", AttributeType.STRING) \
    .start(handle_async_task)
```

### Define a Script Step

```python
from argyll import Client

client = Client("http://localhost:8080/engine")

client.new_step("Double") \
    .required("value", AttributeType.NUMBER) \
    .output("result", AttributeType.NUMBER) \
    .with_script("(* value 2)") \
    .register()
```

### Execute a Flow

```python
from argyll import Client

client = Client("http://localhost:8080/engine")

client.new_flow("greeting-flow-123") \
    .with_goals("greeting") \
    .with_initial_state({"name": "Alice"}) \
    .start()
```

## Features

- **Type-safe builders** - Immutable builder pattern with full type hints
- **Sync and async steps** - Support for both synchronous and asynchronous execution
- **Script steps** - Execute Ale or Lua scripts
- **Flow orchestration** - Define and execute multi-step flows
- **Result memoization** - Cache step results for efficiency
- **Conditional execution** - Use predicates to control step execution
- **Array iteration** - Process arrays with `for_each`

## Development

Install development dependencies:

```bash
make install
```

Run tests with coverage:

```bash
make test-cov
```

Format code:

```bash
make format
```

Run all checks (format, lint, type-check, test):

```bash
make check
```

## API Reference

See the [documentation](docs/) for detailed API reference.

## License

MIT
