# Go Interface

The Go interface provides APIs for building flows and implementing steps in Spuds.

## Quick Start

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

## Guides

- **[Flow Builder](flow-builder.md)** - Create and start flows
- **[Step Builder](step-builder.md)** - Define step specifications

## Installation

```bash
go get github.com/kode4food/spuds/engine/pkg/builder
go get github.com/kode4food/spuds/engine/pkg/api
```

## Environment Variables

- `STEP_PORT` - HTTP server port (default: "8081")
- `STEP_HOSTNAME` - Hostname for endpoint generation (default: "localhost")
