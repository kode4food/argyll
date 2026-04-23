# Argyll Go SDK Step Patterns

Imports:
- github.com/kode4food/argyll/engine/pkg/api
- github.com/kode4food/argyll/sdks/go-builder

Hosted sync HTTP step:
client.NewStep().WithName("Greeting").
    Required("name", api.TypeString).
    Output("greeting", api.TypeString).
    Start(handler)

External GET step:
client.NewStep().WithName("Lookup User").
    Required("user_id", api.TypeString).
    Output("user", api.TypeObject).
    WithMethod("GET").
    WithEndpoint("http://localhost:8081/users/{user_id}").
    Register(ctx)

Async step:
- Add WithAsyncExecution().
- In the handler, create builder.NewAsyncContext(ctx), start background work, and call Success or Fail.
- Return api.NewResult() immediately from the handler.
