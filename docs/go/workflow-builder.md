# Flow Builder

## Basic Usage

```go
import (
    "context"
    "time"

    "github.com/kode4food/spuds/engine/pkg/api"
    "github.com/kode4food/spuds/engine/pkg/builder"
)

client := builder.NewClient("http://localhost:8080", 30*time.Second)

id := builder.NewFlowID("data-pipeline")

err := client.NewFlow(id).
    WithGoals("extract", "transform", "load").
    WithInitialState(api.Args{
        "source": "s3://bucket/data.csv",
        "target": "postgres://db/table",
    }).
    Start(context.Background())
```

## Flow IDs

Use `NewFlowID()` to generate unique IDs:

```go
id := builder.NewFlowID("user-signup")      // "user-signup-a3f2c1"
id := builder.NewFlowID("order-process")    // "order-process-7b4e9d"
```

Sanitizes the prefix (lowercase, spaces to dashes) and appends a 6-character random hex suffix.

## Multiple Goals

```go
// Set all goals at once
client.NewFlow(id).
    WithGoals("step-a", "step-b", "step-c").
    WithInitialState(initState).
    Start(ctx)

// Add goals incrementally
client.NewFlow(id).
    WithGoal("authenticate").
    WithGoal("validate").
    WithGoal("process").
    WithInitialState(initState).
    Start(ctx)

// Direct API
client.StartFlow(ctx, id, []timebox.ID{"step-a", "step-b"}, initState)
```

## Initial State

```go
client.NewFlow(id).
    WithGoals("goal-step").
    WithInitialState(api.Args{
        "user_id": "123",
        "action": "process",
        "config": map[string]any{
            "retry": true,
            "timeout": 30,
        },
    }).
    Start(ctx)
```

## Build Without Starting

```go
request := client.NewFlow(id).
    WithGoals("step-a", "step-b").
    WithInitialState(initState).
    Build()

// Later
err := client.StartFlowWithRequest(ctx, request)
```

## Complete Example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/kode4food/spuds/engine/pkg/api"
    "github.com/kode4food/spuds/engine/pkg/builder"
)

func main() {
    client := builder.NewClient("http://localhost:8080", 30*time.Second)
    id := builder.NewFlowID("data-pipeline")

    err := client.NewFlow(id).
        WithGoals("extract-data", "transform-data", "load-data").
        WithInitialState(api.Args{
            "source": "s3://bucket/data.csv",
            "target": "postgres://db/table",
            "batch_size": 1000,
        }).
        Start(context.Background())

    if err != nil {
        log.Fatalf("Failed to start flow: %v", err)
    }

    log.Printf("Flow started: %s", id)
}
```
