# Workflow Builder

The Workflow builder provides an API for creating and starting workflows.

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

err := client.NewWorkflow("data-pipeline-123").
    WithGoals("extract", "transform", "load").
    WithInitialState(api.Args{
        "source": "s3://bucket/data.csv",
        "target": "postgres://db/table",
    }).
    Start(context.Background())
```

## Creating Workflows

### Simple Workflow

```go
client := builder.NewClient("http://localhost:8080", 30*time.Second)

err := client.NewWorkflow("my-workflow").
    WithGoals("final-step").
    Start(context.Background())
```

### With Initial State

Provide initial arguments that steps can use:

```go
err := client.NewWorkflow("user-signup").
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
err := client.NewWorkflow("multi-goal").
    WithGoals("step-a", "step-b", "step-c").
    Start(context.Background())
```

Or add goals one at a time:

```go
err := client.NewWorkflow("multi-goal").
    WithGoal("step-a").
    WithGoal("step-b").
    WithGoal("step-c").
    Start(context.Background())
```

## Accessing Existing Workflows

Use `Workflow()` to get a client for an existing workflow:

```go
wc := client.Workflow("my-workflow-123")

// Get current state
state, err := wc.GetState(context.Background())
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status: %s\n", state.Status)
fmt.Printf("Attributes: %+v\n", state.Attributes)

// Get workflow ID
flowID := wc.FlowID()
```

## Builder Pattern

The Workflow builder follows an immutable builder pattern:

```go
base := client.NewWorkflow("my-workflow")

// Each method returns a new builder instance
wf1 := base.WithGoals("goal-1")
wf2 := base.WithGoals("goal-2")

// base, wf1, and wf2 are all independent
```

Methods can be chained:

```go
err := client.NewWorkflow("complex-workflow").
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

    // Start a new workflow
    err := client.NewWorkflow("data-pipeline-001").
        WithGoals("extract-data", "transform-data", "load-data").
        WithInitialState(api.Args{
            "source": "s3://bucket/data.csv",
            "target": "postgres://db/table",
            "batch_size": 1000,
        }).
        Start(context.Background())

    if err != nil {
        log.Fatalf("Failed to start workflow: %v", err)
    }

    log.Println("Workflow started successfully")

    // Query workflow state
    wc := client.Workflow("data-pipeline-001")
    state, err := wc.GetState(context.Background())
    if err != nil {
        log.Fatalf("Failed to get workflow state: %v", err)
    }

    log.Printf("Workflow status: %s", state.Status)
    log.Printf("Attributes: %+v", state.Attributes)
}
```

## API Reference

### Client Methods

#### `NewWorkflow(id timebox.ID) *Workflow`
Creates a new workflow builder with the specified ID.

#### `Workflow(flowID timebox.ID) *WorkflowClient`
Returns a client for accessing an existing workflow.

### Workflow Builder Methods

#### `WithGoals(goals ...timebox.ID) *Workflow`
Sets the goal step IDs for the workflow. Replaces any previously set goals.

#### `WithGoal(goal timebox.ID) *Workflow`
Adds a single goal step ID to the workflow.

#### `WithInitialState(init api.Args) *Workflow`
Sets the initial state (arguments) for the workflow.

#### `Start(ctx context.Context) error`
Creates and starts the workflow on the engine.

### WorkflowClient Methods

#### `GetState(ctx context.Context) (*api.WorkflowState, error)`
Retrieves the current state of the workflow.

#### `FlowID() timebox.ID`
Returns the workflow ID for this client.
