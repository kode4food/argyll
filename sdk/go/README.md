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

Run `go generate ./...`, then serve the generated Steps with `gen.Serve(ctx, ArgyllSteps()...)`.

- [Go SDK guide](https://www.argyll.app/docs/sdks/go/)
- [Go Step Generator](https://www.argyll.app/docs/sdks/go-gen/)
- [Runnable examples](../../examples/)

## Develop

```bash
go test ./...
```
