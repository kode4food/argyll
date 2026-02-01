# Argyll Go SDK

Go SDK for building steps and flows with the Argyll distributed orchestrator.

## Installation

```bash
go get github.com/kode4food/argyll/sdks/go-builder
```

## Documentation

- [Step Builder](docs/step-builder.md) - Building and configuring steps
- [Flow Builder](docs/flow-builder.md) - Creating and executing flows
- [Examples](docs/examples.md) - Common patterns and use cases

## Quick Start

### Define a Sync Step

```go
package main

import (
    "context"
    "log"

    "github.com/kode4food/argyll/engine/pkg/api"
    "github.com/kode4food/argyll/sdks/go-builder"
)

func main() {
    client := builder.NewClient("http://localhost:8080")

    handler := func(ctx *builder.StepContext, args api.Args) (api.StepResult, error) {
        name := args["name"].(string)
        return *api.NewResult().WithOutput("greeting", "Hello, "+name), nil
    }

    if err := client.NewStep("Greeting").
        Required("name", api.TypeString).
        Output("greeting", api.TypeString).
        Start(handler); err != nil {
        log.Fatal(err)
    }
}
```

### Define an Async Step

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/kode4food/argyll/engine/pkg/api"
    "github.com/kode4food/argyll/sdks/go-builder"
)

func main() {
    client := builder.NewClient("http://localhost:8080")

    handler := func(ctx *builder.StepContext, args api.Args) (api.StepResult, error) {
        asyncCtx, err := builder.NewAsyncContext(ctx)
        if err != nil {
            return api.StepResult{}, err
        }

        // Start background work
        go func() {
            time.Sleep(5 * time.Second)
            asyncCtx.Success(api.Args{"result": "done"})
        }()

        return *api.NewResult(), nil
    }

    if err := client.NewStep("AsyncTask").
        WithAsyncExecution().
        Output("result", api.TypeString).
        Start(handler); err != nil {
        log.Fatal(err)
    }
}
```

### Define a Script Step

```go
package main

import (
    "context"
    "log"

    "github.com/kode4food/argyll/engine/pkg/api"
    "github.com/kode4food/argyll/sdks/go-builder"
)

func main() {
    client := builder.NewClient("http://localhost:8080")

    script := `
    (define (double x) (* x 2))
    (double value)
    `

    if err := client.NewStep("Double").
        Required("value", api.TypeNumber).
        Output("result", api.TypeNumber).
        WithScript(script).
        Register(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

### Execute a Flow

```go
package main

import (
    "context"
    "log"

    "github.com/kode4food/argyll/engine/pkg/api"
    "github.com/kode4food/argyll/sdks/go-builder"
)

func main() {
    client := builder.NewClient("http://localhost:8080")

    if err := client.NewFlow("greeting-flow-123").
        WithGoals("greeting").
        WithInitialState(api.Args{"name": "Alice"}).
        Start(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## Features

- **Type-safe builders** - Immutable builder pattern with method chaining
- **Sync and async steps** - Support for both synchronous and asynchronous execution
- **Script steps** - Execute Ale or Lua scripts
- **Flow orchestration** - Define and execute multi-step flows
- **Result memoization** - Cache step results for efficiency
- **Conditional execution** - Use predicates to control step execution
- **Array iteration** - Process arrays with for_each

## Builder Pattern

All builders use Go's value semantics for immutability:

```go
builder1 := client.NewStep("Test")
builder2 := builder1.WithID("custom-id")

// builder1 is unchanged
// builder2 has the custom ID
```

## Advanced Features

### Conditional Execution

```go
client.NewStep("ConditionalStep").
    Required("value", api.TypeNumber).
    WithAlePredicate("(> value 10)").
    WithEndpoint("http://localhost:8081/step").
    Register(ctx)
```

### Array Processing

```go
client.NewStep("ProcessItems").
    Required("items", api.TypeArray).
    WithForEach("items").
    Output("processed", api.TypeString).
    WithEndpoint("http://localhost:8081/process").
    Register(ctx)
```

### Result Memoization

```go
client.NewStep("ExpensiveComputation").
    Required("input", api.TypeNumber).
    Output("result", api.TypeNumber).
    WithMemoizable().
    WithEndpoint("http://localhost:8081/compute").
    Register(ctx)
```

### Labels

```go
client.NewStep("DataProcessor").
    WithLabels(api.Labels{"team": "data", "env": "prod"}).
    WithEndpoint("http://localhost:8081/process").
    Register(ctx)
```

## Environment Variables

Configure step server settings:

```bash
export STEP_PORT=8081           # Server port (default: 8081)
export STEP_HOSTNAME=localhost  # Server hostname (default: localhost)
```

## Error Handling

```go
handler := func(ctx *builder.StepContext, args api.Args) (api.StepResult, error) {
    if !authorized {
        return api.StepResult{}, builder.NewHTTPError(401, "Unauthorized")
    }
    // ... process step
}
```

## Testing

```bash
cd builder
go test ./...
```

## Examples

See the [examples](../../examples) directory for complete working examples:
- `simple-step` - Basic synchronous step
- `payment-processor` - Payment processing with validation
- `inventory-resolver` - Inventory lookup
- `notification-sender` - Async notification sending

## API Reference

### Client

- `NewClient(engineURL string) *Client` - Create a new client
- `NewStep(name api.Name) *Step` - Create a step builder
- `NewFlow(flowID api.FlowID) *Flow` - Create a flow builder
- `Flow(flowID api.FlowID) *FlowClient` - Get flow client

### StepBuilder

- `WithID(id string) *Step` - Set custom step ID
- `Required(name, type) *Step` - Add required input
- `Optional(name, type, default) *Step` - Add optional input
- `Const(name, type, value) *Step` - Add const input
- `Output(name, type) *Step` - Declare output
- `WithForEach(name) *Step` - Enable array iteration
- `WithLabel(key, value) *Step` - Add label
- `WithLabels(labels) *Step` - Add multiple labels
- `WithEndpoint(url) *Step` - Set HTTP endpoint
- `WithHealthCheck(url) *Step` - Set health check endpoint
- `WithTimeout(ms) *Step` - Set execution timeout
- `WithScript(script) *Step` - Set Ale script
- `WithScriptLanguage(lang, script) *Step` - Set script with language
- `WithPredicate(lang, script) *Step` - Set predicate
- `WithAsyncExecution() *Step` - Enable async execution
- `WithSyncExecution() *Step` - Enable sync execution
- `WithMemoizable() *Step` - Enable result caching
- `Build() (*api.Step, error)` - Build step
- `Register(ctx) error` - Register step
- `Start(handler) error` - Register and start server

### FlowBuilder

- `WithGoal(stepID) *Flow` - Add single goal
- `WithGoals(...stepIDs) *Flow` - Set all goals
- `WithInitialState(args) *Flow` - Set initial state
- `Start(ctx) error` - Execute flow

### StepContext

- `Context` - Standard Go context
- `Client` - Flow client for operations
- `StepID` - Current step ID
- `Metadata` - Request metadata

### AsyncContext

- `Success(outputs) error` - Mark as successful
- `Fail(err) error` - Mark as failed
- `Complete(result) error` - Complete with full result

## License

MIT
