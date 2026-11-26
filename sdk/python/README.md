# Argyll Python SDK

## Install

```bash
pip install argyll-sdk
```

Define and serve a Step with the fluent builder:

```python
import os

from argyll import Client, StepContext

client = Client(os.environ["ARGYLL_URL"])


def greet(ctx: StepContext, args: dict) -> dict:
    return {"greeting": f"Hello, {args['name']}"}


client.new_step().with_name("Greeting") \
    .required("name") \
    .output("greeting") \
    .start(greet)
```

See the [Python SDK guide](https://www.argyll.app/docs/sdks/python/) for HTTP, async, script, compensation, and Flow examples.

## Develop

```bash
make check
```
