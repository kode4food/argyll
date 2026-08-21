# Argyll Go Step Generator

`argyll-gen` turns ordinary Go functions into Argyll Steps. You write a function; the generator writes the step descriptor, the JSON codecs, the invocation adapter, the panic recovery and the HTTP handlers that speak Argyll's existing synchronous step protocol. The engine cannot tell a generated step from a hand written one.

## Usage

Add a directive to a function and a `go:generate` line to the package:

```go
//go:generate go run github.com/kode4food/argyll/sdk/go/gen/cmd/argyll-gen ./...

type RiskArgs struct {
	CustomerID string
	Amount     int64
}

type RiskResult struct {
	Score    int
	Approved bool
}

//argyll:step
func CalculateRisk(args RiskArgs) (RiskResult, error) {
	return RiskResult{Score: int(args.Amount / 100)}, nil
}
```

Run `go generate ./...` and serve the result:

```go
func main() {
	if err := gen.Serve(context.Background(), ArgyllSteps()...); err != nil {
		log.Fatal(err)
	}
}
```

`Serve` registers every step with the engine and serves its handlers, reading `ARGYLL_ENGINE_URL`, `STEP_HOSTNAME` and `STEP_PORT` from the environment. Use `gen.Register` and `gen.Mux` directly if you want to own the HTTP server.

Generated code lands in `zz_argyll_gen.go` next to the source, and the generated `ArgyllSteps()` returns every `gen.StepDef` in the package.

`zz_argyll_gen.go` is a build artifact rather than source: it is gitignored and rebuilt by `make generate`, which `make check` and `make test` depend on, so it always matches the directives and the generator. Docker builds run `go generate ./...` ahead of `go build` for the same reason.

## Directives

### `//argyll:step`

A function written to be a step. It takes one arguments struct, named or anonymous, and returns an outputs struct, an error, both, or neither:

```go
//argyll:step
func CalculateRisk(args RiskArgs) (RiskResult, error)

//argyll:step
func Greet(args struct{ Name string }) struct{ Greeting string }

//argyll:step
func Audit(args AuditArgs) error
```

Fields of the argument struct become step inputs, fields of the result struct become step outputs. Names default to the Go identifier in `snake_case`, and an [`argyll` field tag](#field-tags) sets the attribute name explicitly.

### `//argyll:wrap`

An ordinary positional function, adapted without changing it. Named parameters and named results supply the attribute names, in `snake_case` like everywhere else:

```go
//argyll:wrap
func CalculateRisk(
	customerID string, amount int64,
) (score int, approved bool, err error)
```

That yields inputs `customer_id` and `amount`, outputs `score` and `approved`.

Either side of the `->` names attributes positionally, overriding what the signature supplies. Go names results only when the function declares them, so naming just the outputs is the common case:

```go
//argyll:wrap -> score, approved
func CalculateRisk(customerID string, amount int64) (int, bool, error)
```

Names in the directive are used verbatim, which makes them an override of the `snake_case` default rather than an input to it. `customer-id` below reaches the flow as `customer-id`:

```go
//argyll:wrap customer-id, amount -> score, approved
func CalculateRisk(customerID string, amount int64) (int, bool, error)
```

The generator checks the arity against the signature at build time, and reports the file and line when they disagree. An omitted side whose parameters or results are unnamed reports the position and asks for the names. The wrapped function knows nothing about Argyll.

### `//argyll:label`

Step labels, one directive per label, in any order alongside a `//argyll:step` or `//argyll:wrap` directive:

```go
//argyll:step
//argyll:label description=send order confirmation notifications
//argyll:label domain=notifications
func NotificationSender(args NotificationArgs) error
```

Everything after the first `=` is the value, spaces included. The generator validates each directive as `key=value`, reporting the file and line otherwise.

The generator emits synchronous Steps.

## The step contract

Go field names map to Argyll attribute names as `snake_case`:

| Go field         | Attribute        |
| ---------------- | ---------------- |
| `CustomerID`     | `customer_id`    |
| `HTTPServer`     | `http_server`    |
| ``Currency string `argyll:"iso_currency"` `` | `iso_currency` |

The function name becomes the step identity, as `kebab-case`: `func CalculateRisk` registers as ID `calculate-risk`, name `Calculate Risk`.

### Field tags

An `argyll` struct tag overrides that default and names the attribute explicitly:

```go
type EnrollArgs struct {
	Currency string `argyll:"iso_currency"`
	Scratch  string `argyll:"-"`
}
```

A tag of `-` keeps the field out of the contract and off the wire entirely, so it stays available as ordinary Go state. Unexported fields are skipped the same way.

The tag applies wherever the struct appears, in inputs, in outputs, and at any nesting depth. Its value is `name` followed by comma separated options, the same shape as `json` tags, so field level options can join it later. The generator rejects an option it does not know, reporting the file, line and field.

Names supplied by a `//argyll:wrap` directive are used verbatim, overriding the `snake_case` default the same way a field tag does.

### Attribute types

Attribute types follow the Go types: strings are `string`, all numeric types are `number`, bools are `boolean`, slices are `array`, structs and maps are `object`. A pointer field is an optional attribute. Structs and string-keyed maps nest to any depth, including recursively:

```go
type Node struct {
	Name     string
	Children []Node
	Next     *Node
}
```

A type that reaches itself, through a slice, a pointer, a map, or a chain of other structs, gets its codec built during package initialization and handed to its users through an indirection, since a self-referential variable initializer is an initialization cycle in Go.

## Failures

An `error` is control plane information, never an output attribute. Returned errors and recovered panics both become step failures over the existing protocol, and stay distinguishable: an error responds `422` with problem details, a panic responds `500` and logs the panic value with its stack. Neither ever appears as step output.

To choose the status yourself, return a `*argyll.HTTPError`, the same type hand written handlers use:

```go
return argyll.NewHTTPError(http.StatusNotFound, "no such customer")
```

Argyll treats every non-2xx response as a failure regardless, so the status is for your own operators and traces.

## Codecs

Argyll's wire format is JSON, but the step contract is not. The generator resolves each Go type into a composition of codecs from the `codec` package, which read and write through `encoding/json/jsontext`:

```go
codec.Struct(
	codec.Field("customer_id", codec.Text[string](), func(v *RiskArgs) *string {
		return &v.CustomerID
	}),
	codec.Field("tags", codec.Slice(codec.Text[string]()), func(v *RiskArgs) *[]string {
		return &v.Tags
	}),
)
```

There is no reflection at runtime, and no bespoke parser per function. `Text`, `Number`, `Boolean`, `Slice`, `Optional`, `Map` and `Struct` compose to cover the supported types. A field type outside that set fails generation with its position and the offending type.
