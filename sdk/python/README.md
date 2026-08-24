# Argyll Python SDK

## Install

```bash
pip install argyll-sdk
```

Define and serve a Step with the fluent builder:

```python
from argyll import AttributeType, Client, StepContext

client = Client("http://localhost:8080")


def greet(ctx: StepContext, args: dict) -> dict:
    return {"greeting": f"Hello, {args['name']}"}


client.new_step().with_name("Greeting") \
    .required("name", AttributeType.STRING) \
    .output("greeting", AttributeType.STRING) \
    .start(greet)
```

See the [Python SDK guide](https://www.argyll.app/docs/sdks/python/) for HTTP, async, script, compensation, and Flow examples.

## Develop

```bash
make check
```
