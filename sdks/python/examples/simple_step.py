"""Simple sync step example."""

from argyll import AttributeType, Client, StepContext

client = Client("http://localhost:8080")


def handle_greeting(ctx: StepContext, args: dict) -> dict:
    """Simple greeting step handler."""
    name = args.get("name", "World")
    greeting = f"Hello, {name}!"
    print(f"Greeting: {greeting}")
    return {"greeting": greeting}


if __name__ == "__main__":
    client.new_step().with_name("Greeting") \
        .required("name", AttributeType.STRING) \
        .output("greeting", AttributeType.STRING) \
        .with_label("category", "demo") \
        .start(handle_greeting)
