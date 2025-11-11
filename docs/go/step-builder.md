# Step Builder

## Basic Usage

```go
package main

import (
    "github.com/kode4food/spuds/engine/pkg/api"
    "github.com/kode4food/spuds/engine/pkg/builder"
)

func main() {
    builder.SetupStep("User Resolver", buildStep, handleStep)
}

func buildStep(s *builder.Step) *builder.Step {
    return s.
        Required("user_id", api.TypeString).
        Output("user_name", api.TypeString).
        Output("user_email", api.TypeString)
}

func handleStep(sctx *builder.StepContext) (api.StepResult, error) {
    userID := sctx.GetString("user_id")
    userName, userEmail := lookupUser(userID)

    return *api.NewResult().
        WithOutput("user_name", userName).
        WithOutput("user_email", userEmail), nil
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

Get webhook URL from metadata:

```go
func handleStep(sctx *builder.StepContext) (api.StepResult, error) {
    webhookURL := sctx.Metadata()["webhook_url"].(string)
    go processAsync(webhookURL, sctx.Args())
    return *api.NewResult(), nil
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

func handleOrderProcessor(sctx *builder.StepContext) (api.StepResult, error) {
    orderID := sctx.GetString("order_id")
    items := sctx.Get("items")

    sctx.Logger().Info("processing order", "order_id", orderID)

    total := calculateTotal(items)

    return *api.NewResult().
        WithOutput("total_amount", total).
        WithOutput("processed_at", time.Now().Unix()), nil
}

func calculateTotal(items any) float64 {
    return 99.99
}
```
