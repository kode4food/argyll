# Go SDK Examples

These examples assume the engine is running at http://localhost:8080.

## Register a script step

```go
client := builder.NewClient("http://localhost:8080", 30*time.Second)

err := client.NewStep("Hello Script").
    Required("name", api.TypeString).
    Output("greeting", api.TypeString).
    WithScript("{:greeting name}").
    Register(context.Background())
if err != nil {
    log.Fatal(err)
}
```

## Start a flow

```go
flowID := builder.NewFlowID("hello-flow")

err := client.NewFlow(flowID).
    WithGoal("hello-script").
    WithInitialState(api.Args{"name": "Argyll"}).
    Start(context.Background())
if err != nil {
    log.Fatal(err)
}
```

## Query flow state

```go
fc := client.Flow(flowID)
state, err := fc.GetState(context.Background())
if err != nil {
    log.Fatal(err)
}
log.Printf("status=%s attrs=%v", state.Status, state.Attributes)
```
