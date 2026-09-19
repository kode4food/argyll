# Argyll Go SDK

## Install

```bash
go get github.com/kode4food/argyll/sdk/go
```

Write an ordinary function, mark it as a Step, and generate its contract and HTTP adapter:

```go
//go:generate go run github.com/kode4food/argyll/sdk/go/gen/cmd/argyll-gen ./...

type (
	GreetArgs struct {
		Name string
	}
	GreetRes struct {
		Greeting string
	}
)

//argyll:step
func Greet(args GreetArgs) GreetRes {
	return GreetRes{Greeting: "Hello, " + args.Name}
}
```

Run `go generate ./...`, then serve the generated Steps with `gen.Serve(ctx, ArgyllServiceSteps()...)`.

For a standalone `package main`, pass `-server` before the package pattern to generate a minimal `main` function that registers and serves the steps. Generated servers log each step invocation with `slog`; normal generation adds no logging:

```go
//go:generate go run github.com/kode4food/argyll/sdk/go/gen/cmd/argyll-gen -server .
```

Normal generation also runs the same functions inside an embedded engine: install `ArgyllEmbeddedHandlers()` among the engine's handlers and register `ArgyllEmbeddedSteps()`. The generated handlers convert values in process, with no HTTP or JSON between the engine and your function.

- [Go SDK guide](https://www.argyll.app/docs/sdks/go/)
- [Go Step Generator](https://www.argyll.app/docs/sdks/go-gen/)
- [Embedding](https://www.argyll.app/docs/guides/embedding/)
- [Runnable examples](../../examples/)

## Develop

```bash
go test ./...
```
