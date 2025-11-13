# Step Builder

## Basic Usage

#### Register Only (no server)

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

#### Register + Start Server

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

    handler := func(ctx context.Context, args api.Args) (api.StepResult, error) {
        userID, _ := args["user_id"].(string)
        // Lookup user...
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

Use client to create async context:

```go
func main() {
    client := builder.NewClient(engineURL, 30*time.Second)
    handler := makeHandler(client)
    builder.SetupStep("Async Step", build, handler)
}

func makeHandler(client *builder.Client) api.StepHandler {
    return func(ctx context.Context, args api.Args) (api.StepResult, error) {
        async, err := client.NewAsyncContext(ctx)
        if err != nil {
            return *api.NewResult().WithError(err), nil
        }

        go func() {
            result := processAsync(args)
            if err := async.Success(api.Args{"output": result}); err != nil {
                slog.Error("webhook failed", "error", err)
            }
        }()

        return api.StepResult{Success: true}, nil
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

Execute conditionally based on workflow state:

```go
// Ale
s.WithAlePredicate(`(> amount 100)`)

// Lua
s.WithLuaPredicate(`return status == "active"`)

// Custom
s.WithPredicate("ale", `(> amount 100)`)
```

## SetupStep

```go
func SetupStep(
    name api.Name,
    build func(*Step) *Step,
    handle StepHandlerFunc,
) error
```

Automatically:
1. Builds step definition
2. Generates HTTP endpoint
3. Registers with engine (with retry)
4. Starts HTTP server
5. Sets up panic recovery

### Environment Variables

- `SPUDS_ENGINE_URL` - Engine URL (default: "http://localhost:8080")
- `STEP_PORT` - Server port (default: "8081")
- `STEP_HOSTNAME` - Hostname (default: "localhost")

## Complete Example

```go
package main

import (
    "time"

    "github.com/kode4food/spuds/engine/pkg/api"
    "github.com/kode4food/spuds/engine/pkg/builder"
)

func main() {
    builder.SetupStep("Order Processor", buildOrderProcessor, handleOrderProcessor)
}

func buildOrderProcessor(s *builder.Step) *builder.Step {
    return s.
        WithVersion("1.0.0").
        WithTimeout(30 * api.Second).
        Required("order_id", api.TypeString).
        Required("items", api.TypeArray).
        Optional("priority", api.TypeString, `"normal"`).
        Output("total_amount", api.TypeNumber).
        Output("processed_at", api.TypeNumber).
        WithAlePredicate(`(> (len items) 0)`)
}

func handleOrderProcessor(ctx context.Context, args api.Args) (api.StepResult, error) {
    orderID, _ := args["order_id"].(string)
    items := args["items"]

    slog.Info("processing order", "order_id", orderID)

    total := calculateTotal(items)

    return *api.NewResult().
        WithOutput("total_amount", total).
        WithOutput("processed_at", time.Now().Unix()), nil
}

func calculateTotal(items any) float64 {
    return 99.99
}
```
