# Argyll Python SDK Step Patterns

Imports:
- from argyll import Client, StepContext

Hosted sync HTTP step:
client.new_step().with_name("Greeting") \
    .required("name") \
    .output("greeting") \
    .start(handle_greeting)

Sync step backed by an existing HTTP GET endpoint:
client.new_step().with_name("Lookup User") \
    .required("user_id") \
    .output("user") \
    .with_method("GET") \
    .with_endpoint("http://localhost:8081/users/{user_id}") \
    .register()

Async step:
- Add with_async_invoke().
- In the handler, create AsyncContext from the webhook URL in ctx.metadata, which is populated from the Argyll-Webhook-URL header, start background work, and call success or fail.
- Return {} immediately from the handler.
