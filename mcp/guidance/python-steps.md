# Argyll Python SDK Step Patterns

Imports:
- from argyll import Client, StepContext, AttributeType, StepResult

Hosted sync HTTP step:
client.new_step().with_name("Greeting") \
    .required("name", AttributeType.STRING) \
    .output("greeting", AttributeType.STRING) \
    .start(handle_greeting)

External GET step:
client.new_step().with_name("Lookup User") \
    .required("user_id", AttributeType.STRING) \
    .output("user", AttributeType.OBJECT) \
    .with_method("GET") \
    .with_endpoint("http://localhost:8081/users/{user_id}") \
    .register()

Async step:
- Add with_async_execution().
- In the handler, create AsyncContext from the webhook URL in ctx.metadata, start background work, and call success or fail.
- Return StepResult(success=True, outputs={}) immediately from the handler.
