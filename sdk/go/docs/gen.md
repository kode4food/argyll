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

One `go:generate` line is enough for a whole tree. The generator scans every file of every package matching the pattern, so `./...` reaches the directive's own package and everything beneath it, writing a `zz_argyll_gen.go` into each package that declares Steps. Extra `go:generate` lines are safe: a second pass over the same package produces the same bytes and leaves the file alone.

`zz_argyll_gen.go` is a build artifact rather than source: it is gitignored and rebuilt by `make generate`, which `make check` and `make test` depend on, so it always matches the directives and the generator. Docker builds run `go generate ./...` ahead of `go build` for the same reason.

## Directives

Directives and field tags read the same way: a leading value names the thing, and semicolon separated `key:value` properties configure it. Whitespace anywhere in that is ignored.

```go
//argyll:step   charge-card-v2;name:Charge Card (v2)
//argyll:wrap   score-v2(customer-id, amount) -> (score, approved)
//argyll:props  timeout: 2500; predicate: return args.amount > 0
//argyll:labels domain: risk; tier: gold
```

```go
Currency string `argyll:"iso_currency;role:optional;default:USD"`
```

`props` and `labels` take properties only, and both repeat, so a long set spreads across lines. Only `//argyll:wrap` takes an attribute spec: a `//argyll:step` or a field tag carrying one is an error rather than a silent no-op.

An omitted ID is the function name in `kebab-case`. The ID also names the generated `StepDef` var, so it stays `kebab-case`: lowercase letters and digits, hyphen separated.

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
//argyll:wrap -> (score, approved)
func CalculateRisk(customerID string, amount int64) (int, bool, error)
```

Names in the directive are used verbatim, which makes them an override of the `snake_case` default rather than an input to it. `customer-id` below reaches the flow as `customer-id`, and `score-v2` is the step ID:

```go
//argyll:wrap score-v2(customer-id, amount) -> (score, approved)
func CalculateRisk(customerID string, amount int64) (int, bool, error)
```

An empty list is empty rather than inferred, so `() -> (score)` declares a step with no inputs.

The generator checks the arity against the signature at build time, and reports the file and line when they disagree. An omitted side whose parameters or results are unnamed reports the position and asks for the names. The wrapped function knows nothing about Argyll.

### `//argyll:props`

Step properties, continuing the ones on the `//argyll:step` or `//argyll:wrap` line:

```go
//argyll:step charge-card
//argyll:props name: Charge Card; timeout: 2500
//argyll:props predicate: return args.amount > 0
func ChargeCard(args ChargeArgs) (ChargeResult, error)
```

| Property    | Effect                                                       |
| ----------- | ------------------------------------------------------------ |
| `name`      | Display name, defaulting to the function name in `Title Case` |
| `memoize`   | `true` to memoize the step                                    |
| `timeout`   | Invocation timeout in milliseconds                            |
| `predicate` | Lua predicate gating the step                                 |

Everything after the first `:` is the value, spaces and further colons included, so a Lua predicate needs no quoting. A value ends at the next `;`.

Every property here is engine side: the engine memoizes, times out and evaluates predicates on its own, and the handler never learns it happened. Generated steps are synchronous and answer `POST`, so there is nothing to configure on either.

### `//argyll:labels`

Step labels, in the same repeatable form:

```go
//argyll:step
//argyll:labels description: send order confirmation notifications
//argyll:labels domain: notifications; tier: gold
func NotificationSender(args NotificationArgs) error
```

### Validation

The generator assembles an `api.Step` and runs the engine's own `Validate` on it before writing anything, so a step the engine would reject fails `go generate` with the file and line that declared it. The generated file carries that validated step in the wire form the engine receives, and registration adds only the host the step server is reachable on.

## The step contract

Go field names map to Argyll attribute names as `snake_case`:

| Go field         | Attribute        |
| ---------------- | ---------------- |
| `CustomerID`     | `customer_id`    |
| `HTTPServer`     | `http_server`    |
| ``Currency string `argyll:"iso_currency"` `` | `iso_currency` |

The function name becomes the step identity, as `kebab-case`: `func CalculateRisk` registers as ID `calculate-risk`, name `Calculate Risk`. The [directive](#directives) sets either of those explicitly.

### Field tags

An `argyll` struct tag overrides that default and names the attribute explicitly:

```go
type EnrollArgs struct {
	Currency string `argyll:"iso_currency"`
	Scratch  string `argyll:"-"`
}
```

A tag of `-` keeps the field out of the contract and off the wire entirely, so it stays available as ordinary Go state. Unexported fields are skipped the same way.

The tag applies wherever the struct appears, in inputs, in outputs, and at any nesting depth. The generator rejects a property it does not know, reporting the file, line and field.

### Attribute properties

Properties after the name configure the attribute. Leave the name off to keep the `snake_case` default and still set properties:

```go
type ChargeArgs struct {
	OrderID  string `argyll:"for_each:true;collect:all"`
	Note     string `argyll:"role:optional"`
	Currency string `argyll:"role:optional;default:USD;deadline:5000"`
	Gateway  string `argyll:"role:const;value:stripe"`
	FlowID   string `argyll:"flow;role:meta;key:flow_id"`
	Amount   int64  `argyll:"match:return args.amount > 0"`
}
```

| Property   | Effect                                                        |
| ---------- | ------------------------------------------------------------- |
| `role`     | `required`, `optional`, `const` or `meta`, defaulting to `required` for a value field and `optional` for a pointer |
| `default`  | Default value of an optional input                             |
| `value`    | Fixed value of a const input                                   |
| `key`      | Execution metadata key filling a meta input                    |
| `collect`  | `first`, `last`, `all`, `some` or `none`                       |
| `deadline` | Collection deadline of an optional input, in milliseconds      |
| `for_each` | `true` to expand the attribute into one work item per element  |
| `match`    | Lua match gate on a required input                             |
| `mapping`  | Name the attribute is mapped to                                |

Each property belongs to a role, and using one against the wrong role is an error naming both: `default` needs `optional`, `value` needs `const`, `key` needs `meta`, `match` needs `required`.

`default` and `value` reach the engine as JSON, and the generator quotes the value of a string attribute for you, so `default:USD` is written plainly.

A `for_each` attribute is declared as an array and arrives one element at a time, so the Go field carries the element type: `OrderID string` above declares `order_id` as an array and receives a single order ID per work item.

Properties belong to the attributes of a step, which are the fields of its arguments and outputs structs. An output takes a name and a `mapping`, nothing else. Fields nested inside those structs are values within an attribute rather than attributes themselves, so only their name applies. A `//argyll:wrap` step names its attributes in the directive rather than in a struct, and those names are used verbatim, overriding the `snake_case` default the same way a field tag does.

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
