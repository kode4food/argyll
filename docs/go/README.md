# Go Interface

The Go interface provides APIs for building workflows and implementing steps in Spuds.

## Quick Start

```go
package main

import (
    "github.com/kode4food/spuds/engine/pkg/api"
    "github.com/kode4food/spuds/engine/pkg/builder"
)

func main() {
    builder.SetupStep("Greet User", buildStep, handleStep)
}

func buildStep(s *builder.Step) *builder.Step {
    return s.
        Required("name", api.TypeString).
        Output("greeting", api.TypeString)
}

func handleStep(sctx *builder.StepContext) (api.StepResult, error) {
    name := sctx.GetString("name")
    greeting := "Hello, " + name + "!"

    return *api.NewResult().WithOutput("greeting", greeting), nil
}
```

## Guides

- **[Workflow Builder](workflow-builder.md)** - Create and start workflows
- **[Step Builder](step-builder.md)** - Define step specifications
- **[Step Handler](step-handler.md)** - Implement step logic with StepContext

## Installation

```bash
go get github.com/kode4food/spuds/engine/pkg/builder
go get github.com/kode4food/spuds/engine/pkg/api
```

## Environment Variables

- `SPUDS_ENGINE_URL` - Engine URL (default: "http://localhost:8080")
- `STEP_PORT` - HTTP server port (default: "8081")
- `STEP_HOSTNAME` - Hostname for endpoint generation (default: "localhost")
