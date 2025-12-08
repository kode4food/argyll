# Go Interface

The Go interface provides APIs for building flows and implementing steps in Argyll.

## Quick Start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/kode4food/argyll/engine/pkg/api"
    "github.com/kode4food/argyll/engine/pkg/builder"
)

func main() {
    client := builder.NewClient("http://localhost:8080", 30*time.Second)

    handler := func(ctx *builder.StepContext, args api.Args) (api.StepResult, error) {
        name, _ := args["name"].(string)
        greeting := "Hello, " + name + "!"

        // Access to flow client through context
        // ctx.Client can be used to query flow state or start new flows

        return *api.NewResult().WithOutput("greeting", greeting), nil
    }

    err := client.NewStep("Greet User").
        Required("name", api.TypeString).
        Output("greeting", api.TypeString).
        Start(handler)

    if err != nil {
        log.Fatal(err)
    }
}
```

## Common Operations

### List Registered Steps

Query all steps registered with the orchestrator:

```go
steps, err := client.ListSteps(context.Background())
if err != nil {
    log.Fatal(err)
}

for _, step := range steps.Steps {
    log.Printf("Step: %s (ID: %s, Type: %s)",
        step.Name, step.ID, step.Type)
}
```

### Generate Unique Flow IDs

Create flow IDs with a descriptive prefix and unique suffix:

```go
// Generates IDs like "order-processor-a3f2d9"
flowID := builder.NewFlowID("Order Processor")

flow := client.NewFlow(flowID).
    WithGoal("process-order").
    WithInitialState(api.Args{"order_id": "12345"})
```

## Guides

- **[Flow Builder](flow-builder.md)** - Create and start flows
- **[Step Builder](step-builder.md)** - Define step specifications

## Installation

```bash
go get github.com/kode4food/argyll/engine/pkg/builder
go get github.com/kode4food/argyll/engine/pkg/api
```

## Environment Variables

- `STEP_PORT` - HTTP server port (default: "8081")
- `STEP_HOSTNAME` - Hostname for endpoint generation (default: "localhost")
