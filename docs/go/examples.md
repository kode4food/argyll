# Go SDK Examples

These examples assume the engine is running at http://localhost:8080.

## Setup

```go
client := builder.NewClient("http://localhost:8080", 30*time.Second)
```

## Register a sync HTTP step

```go
err := client.NewStep("Transform Order").
    Required("order_id", api.TypeString).
    Output("status", api.TypeString).
    WithEndpoint("http://localhost:8081/transform").
    Register(context.Background())
if err != nil {
    log.Fatal(err)
}
```

## Register an async HTTP step

```go
err := client.NewStep("Send Email").
    Required("user_id", api.TypeString).
    Output("sent", api.TypeBool).
    WithEndpoint("http://localhost:8082/send").
    WithAsyncExecution().
    Register(context.Background())
if err != nil {
    log.Fatal(err)
}
```

## Register a script step

```go
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

## Wait for completion (polling)

```go
deadline := time.Now().Add(10 * time.Second)
for {
    state, err = fc.GetState(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    if state.Status == api.FlowCompleted || state.Status == api.FlowFailed {
        break
    }
    if time.Now().After(deadline) {
        log.Fatal("timed out waiting for flow completion")
    }
    time.Sleep(200 * time.Millisecond)
}
```
