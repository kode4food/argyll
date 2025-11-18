# Flow Builder

The Flow builder provides an API for creating and starting flows.

## Basic Usage

```go
import (
    "context"
    "time"

    "github.com/kode4food/timebox"
    "github.com/kode4food/spuds/engine/pkg/api"
    "github.com/kode4food/spuds/engine/pkg/builder"
)

client := builder.NewClient("http://localhost:8080", 30*time.Second)

err := client.NewFlow("data-pipeline-123").
    WithGoals("extract", "transform", "load").
    WithInitialState(api.Args{
        "source": "s3://bucket/data.csv",
        "target": "postgres://db/table",
    }).
    Start(context.Background())
```

## Creating Flows

### Simple Flow

```go
client := builder.NewClient("http://localhost:8080", 30*time.Second)

err := client.NewFlow("my-flow").
    WithGoals("final-step").
    Start(context.Background())
```

### With Initial State

Provide initial arguments that steps can use:

```go
err := client.NewFlow("user-signup").
    WithGoals("send-welcome-email").
    WithInitialState(api.Args{
        "user_id": "12345",
        "email": "user@example.com",
        "name": "John Doe",
    }).
    Start(context.Background())
```

### Multiple Goals

Specify multiple goal steps:

```go
err := client.NewFlow("multi-goal").
    WithGoals("step-a", "step-b", "step-c").
    Start(context.Background())
```

Or add goals one at a time:

```go
err := client.NewFlow("multi-goal").
    WithGoal("step-a").
    WithGoal("step-b").
    WithGoal("step-c").
    Start(context.Background())
```

## Accessing Existing Flows

Use `Flow()` to get a client for an existing flow:

```go
wc := client.Flow("my-flow-123")

// Get current state
state, err := wc.GetState(context.Background())
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status: %s\n", state.Status)
fmt.Printf("Attributes: %+v\n", state.Attributes)

// Get flow ID
flowID := wc.FlowID()
```

## Builder Pattern

The Flow builder follows an immutable builder pattern:

```go
base := client.NewFlow("my-flow")

// Each method returns a new builder instance
wf1 := base.WithGoals("goal-1")
wf2 := base.WithGoals("goal-2")

// base, wf1, and wf2 are all independent
```

Methods can be chained:

```go
err := client.NewFlow("complex-flow").
    WithGoals("final-step").
    WithInitialState(api.Args{
        "config": map[string]any{
            "retry": true,
            "timeout": 30,
        },
        "user_id": "123",
    }).
    Start(context.Background())
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

    // Start a new flow
    err := client.NewFlow("data-pipeline-001").
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

    log.Println("Flow started successfully")

    // Query flow state
    wc := client.Flow("data-pipeline-001")
    state, err := wc.GetState(context.Background())
    if err != nil {
        log.Fatalf("Failed to get flow state: %v", err)
    }

    log.Printf("Flow status: %s", state.Status)
    log.Printf("Attributes: %+v", state.Attributes)
}
```

## API Reference

### Client Methods

#### `NewFlow(id timebox.ID) *Flow`
Creates a new flow builder with the specified ID.

#### `Flow(flowID timebox.ID) *FlowClient`
Returns a client for accessing an existing flow.

### Flow Builder Methods

#### `WithGoals(goals ...timebox.ID) *Flow`
Sets the goal step IDs for the flow. Replaces any previously set goals.

#### `WithGoal(goal timebox.ID) *Flow`
Adds a single goal step ID to the flow.

#### `WithInitialState(init api.Args) *Flow`
Sets the initial state (arguments) for the flow.

#### `Start(ctx context.Context) error`
Creates and starts the flow on the engine.

### FlowClient Methods

#### `GetState(ctx context.Context) (*api.FlowState, error)`
Retrieves the current state of the flow.

#### `FlowID() timebox.ID`
Returns the flow ID for this client.
