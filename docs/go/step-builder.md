# Step Builder

The Step builder provides an API for defining and registering steps.

## Basic Usage

### Register Only (no server)

For registering script steps or when you're running your own HTTP server:

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

    // Script step - no server needed
    err := client.NewStep("Text Formatter").
        Required("text", api.TypeString).
        Output("formatted_text", api.TypeString).
        WithScript(`{:formatted_text (str "Hello, " text)}`).
        Register(context.Background())

    if err != nil {
        log.Fatal(err)
    }

    // HTTP step with external server
    err = client.NewStep("User Resolver").
        Required("user_id", api.TypeString).
        Output("user_name", api.TypeString).
        WithEndpoint("http://localhost:8081/user-resolver").
        Register(context.Background())

    if err != nil {
        log.Fatal(err)
    }
}
```

### Register + Start Server

For HTTP steps where you want the builder to create and start the server:

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

    handler := func(ctx *builder.StepContext, args api.Args) (api.StepResult, error) {
        userID, _ := args["user_id"].(string)
        // Lookup user...

        // Access to flow context
        // ctx.Client provides flow client for queries
        // ctx.StepID contains the current step ID
        // ctx.Metadata contains workflow metadata

        return *api.NewResult().
            WithOutput("user_name", "John Doe").
            WithOutput("user_email", "john@example.com"), nil
    }

    err := client.NewStep("User Resolver").
        Required("user_id", api.TypeString).
        Output("user_name", api.TypeString).
        Output("user_email", api.TypeString).
        Start(handler)

    if err != nil {
        log.Fatal(err)
    }
}
```

## Attributes

### Required

```go
s.Required("user_id", api.TypeString).
  Required("amount", api.TypeNumber).
  Required("items", api.TypeArray)
```

### Optional

Default values must be valid JSON:

```go
s.Optional("priority", api.TypeString, `"normal"`).
  Optional("retry_count", api.TypeNumber, "3").
  Optional("enabled", api.TypeBoolean, "true")
```

### Output

```go
s.Output("result", api.TypeString).
  Output("status_code", api.TypeNumber).
  Output("response_data", api.TypeObject)
```

### Available Types

- `api.TypeString` - JSON string
- `api.TypeNumber` - JSON number
- `api.TypeBoolean` - JSON boolean
- `api.TypeObject` - JSON object
- `api.TypeArray` - JSON array
- `api.TypeNull` - JSON null
- `api.TypeAny` - Any JSON type

## Execution Types

### Synchronous

Completes within the HTTP request:

```go
s.WithSyncExecution().
  Required("input", api.TypeString).
  Output("output", api.TypeString)
```

### Asynchronous

Returns immediately and posts results via webhook:

```go
s.WithAsyncExecution().
  Required("input", api.TypeString).
  Output("output", api.TypeString)
```

Use StepContext to create async context:

```go
func main() {
    client := builder.NewClient("http://localhost:8080", 30*time.Second)

    handler := func(ctx *builder.StepContext, args api.Args) (api.StepResult, error) {
        async, err := builder.NewAsyncContext(ctx)
        if err != nil {
            return *api.NewResult().WithError(err), nil
        }

        go func() {
            result := processAsync(args)
            if err := async.Success(api.Args{"output": result}); err != nil {
                log.Printf("webhook failed: %v", err)
            }
        }()

        return api.StepResult{Success: true}, nil
    }

    err := client.NewStep("Async Step").
        WithAsyncExecution().
        Required("input", api.TypeString).
        Output("output", api.TypeString).
        Start(handler)

    if err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

### Custom Step ID

Default: generated from name ("User Resolver" → "user-resolver")

```go
s.WithID("custom-step-id")
```

### Version

```go
s.WithVersion("2.1.0")
```

### Timeout

In milliseconds:

```go
s.WithTimeout(60 * api.Second)
```

## Advanced

### ForEach

Process array items in parallel:

```go
s.Required("users", api.TypeArray).
  WithForEach("users").
  Output("results", api.TypeArray)
```

Engine splits arrays into work items, executes in parallel, and aggregates results.

### Predicates

Execute conditionally based on flow state:

```go
// Ale
s.WithAlePredicate(`(> amount 100)`)

// Lua
s.WithLuaPredicate(`return status == "active"`)

// Custom
s.WithPredicate("ale", `(> amount 100)`)
```

### Updating Steps

Mark a step as modified to update the existing registration:

```go
err := client.NewStep("User Resolver").
    Required("user_id", api.TypeString).
    Output("user_name", api.TypeString).
    Output("user_email", api.TypeString).
    Output("user_age", api.TypeNumber). // New output
    Update(). // Mark as dirty
    Start(handler)
```

## Environment Variables

- `STEP_PORT` - Server port (default: "8081")
- `STEP_HOSTNAME` - Hostname (default: "localhost")

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

    handler := func(ctx *builder.StepContext, args api.Args) (api.StepResult, error) {
        orderID, _ := args["order_id"].(string)
        items := args["items"]

        log.Printf("processing order: %s", orderID)
        log.Printf("step ID: %s", ctx.StepID)

        // Can access flow state if needed
        if ctx.Client.FlowID() != "" {
            state, _ := ctx.Client.GetState(ctx)
            log.Printf("flow status: %v", state.Status)
        }

        total := calculateTotal(items)

        return *api.NewResult().
            WithOutput("total_amount", total).
            WithOutput("processed_at", time.Now().Unix()), nil
    }

    err := client.NewStep("Order Processor").
        WithVersion("1.0.0").
        WithTimeout(30 * api.Second).
        Required("order_id", api.TypeString).
        Required("items", api.TypeArray).
        Optional("priority", api.TypeString, `"normal"`).
        Output("total_amount", api.TypeNumber).
        Output("processed_at", api.TypeNumber).
        WithAlePredicate(`(> (length items) 0)`).
        Start(handler)

    if err != nil {
        log.Fatal(err)
    }
}

func calculateTotal(items any) float64 {
    return 99.99
}
```

## API Reference

### Client Methods

#### `NewStep(name api.Name) *Step`
Creates a new step builder with the specified name.

### Step Builder Methods

#### Attribute Definition
- `Required(name api.Name, argType api.AttributeType) *Step` - Add required input
- `Optional(name api.Name, argType api.AttributeType, defaultValue string) *Step` - Add optional input with default
- `Output(name api.Name, argType api.AttributeType) *Step` - Add output

#### Configuration
- `WithID(id string) *Step` - Set custom step ID
- `WithVersion(version string) *Step` - Set step version
- `WithTimeout(timeout int64) *Step` - Set execution timeout (milliseconds)
- `WithEndpoint(endpoint string) *Step` - Set HTTP endpoint
- `WithHealthCheck(endpoint string) *Step` - Set health check endpoint

#### Execution Type
- `WithSyncExecution() *Step` - Mark as synchronous
- `WithAsyncExecution() *Step` - Mark as asynchronous
- `WithScriptExecution() *Step` - Mark as script-based

#### Scripts
- `WithScript(script string) *Step` - Add Ale script
- `WithScriptLanguage(lang, script string) *Step` - Add script with custom language
- `WithAlePredicate(script string) *Step` - Add Ale predicate
- `WithLuaPredicate(script string) *Step` - Add Lua predicate
- `WithPredicate(language, script string) *Step` - Add custom predicate

#### Advanced
- `WithForEach(name api.Name) *Step` - Enable parallel array processing

#### Registration
- `Build() (*api.Step, error)` - Build step definition
- `Register(ctx context.Context) error` - Register step with engine
- `Update() *Step` - Mark step as modified (for updates)
- `Start(handler builder.StepHandler) error` - Register and start HTTP server

### StepHandler Type

```go
type StepHandler func(*StepContext, api.Args) (api.StepResult, error)
```

### StepContext Type

```go
type StepContext struct {
    context.Context                    // Standard Go context
    Client   *FlowClient               // Flow client for queries
    StepID   StepID                    // Current step ID
    Metadata api.Metadata              // Workflow metadata
}
```
