"""Simple sync step example."""

from argyll import Client, StepContext, AttributeType, StepResult

client = Client("http://localhost:8080/engine")


def handle_greeting(ctx: StepContext, args: dict) -> StepResult:
    """Simple greeting step handler."""
    name = args.get("name", "World")
    greeting = f"Hello, {name}!"
    print(f"Greeting: {greeting}")
    return StepResult(success=True, outputs={"greeting": greeting})


if __name__ == "__main__":
    client.new_step("Greeting") \
        .required("name", AttributeType.STRING) \
        .output("greeting", AttributeType.STRING) \
        .with_label("category", "demo") \
        .start(handle_greeting)
