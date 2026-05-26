# Argyll Go SDK Step Patterns

Imports:
- github.com/kode4food/argyll/engine/pkg/api
- github.com/kode4food/argyll/sdks/go-builder

Hosted sync HTTP step:
client.NewStep().WithName("Greeting").
    Required("name", api.TypeString).
    Output("greeting", api.TypeString).
    Start(handler)

Sync step backed by an existing HTTP GET endpoint:
client.NewStep().WithName("Lookup User").
    Required("user_id", api.TypeString).
    Output("user", api.TypeObject).
    WithMethod("GET").
    WithEndpoint("http://localhost:8081/users/{user_id}").
    Register(ctx)

Async step:
- Add WithAsyncExecution().
- In the handler, create builder.NewAsyncContext(ctx), start background work, and call Success or Fail.
- Return api.Args{} immediately from the handler.
