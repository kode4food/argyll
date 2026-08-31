# Go Style Guide

## Naming Conventions

### Receiver Names

Single lowercase letter, first letter of type name:

```go
// Good
func (e *Engine) Start() {}
func (tx *flowTx) process() {}
func (s *Store) Get() {}
func (c *Client) Do() {}

// Bad
func (engine *Engine) Start() {}
func (self *Engine) Start() {}
func (this *Engine) Start() {}
```

### Variable Names

**Prefer short names.** The closer a variable is used to where it's declared, the shorter it can be. Loop variables can be single letters.

```go
// Good - short names, close usage
for i, s := range steps {
    if ok := validate(s); !ok {
        continue
    }
}

for _, e := range events {
    process(e)
}

// Good - map access always uses 'ok'
if v, ok := cache[key]; ok {
    return v
}

if step, ok := flow.Steps[id]; ok {
    return step.Status
}

// Bad - verbose names for tight scope
for index, currentStep := range steps {
    if exists := validate(currentStep); !exists {  // Use 'ok', not 'exists'
        continue
    }
}
```

Name a local for its semantic subject rather than its type, and use the subject alone rather than the full noun phrase:

```go
// Good
for id := range pl.Steps {
    health := r.resolve(id)
    if health.Status == api.HealthUnhealthy {
        return flowStepHealth(id, health)
    }
}

cat, err := eng.GetCatalogState()
flow, err := eng.GetFlowState(flowID)
work := exec.WorkItems[token]
h, ok := st.Health[stepID]

// Bad
for plannedStepID := range executionPlan.Steps {
    plannedStepHealth := r.resolve(plannedStepID)
    if plannedStepHealth.Status == api.HealthUnhealthy {
        return flowStepHealth(plannedStepID, plannedStepHealth)
    }
}

catState, err := eng.GetCatalogState()
flowState, err := eng.GetFlowState(flowID)
workItem := exec.WorkItems[token]
stepHealth, ok := st.Health[stepID]
```

Use longer names only when the broader scope really needs them, such as struct fields, exported APIs, or tests where the variable itself is the subject under test.

**Longer names for wider scope** (exported functions, struct fields):

```go
// Good - clear at API boundaries
func (e *Engine) StartFlow(
    flowID api.FlowID, goalSteps []api.StepID, initState api.Args,
) (api.FlowState, error)

// Good - descriptive struct fields
type FlowState struct {
    FlowID     api.FlowID
    Status     FlowStatus
    Executions map[StepID]*Execution
}
```

**Idiomatic short names**:

These short forms apply to **variables only**: local variables, function parameters, and range/closure bindings. They never apply to struct fields, which are semantic identifiers read by every caller and must keep their full descriptive names (`step`, `flow`, `space`, `plan`, `catalog`), whether the struct is exported, unexported, an `Args`/`Res` bundle, or an anonymous table-test struct. The same goes for type names, function names, method names, and map keys. Renaming a field is never part of applying this rule.

| Name                   | Usage                                       |
| ---------------------- | ------------------------------------------- |
| `i`, `j`, `k`          | Loop indices                                |
| `n`                    | Count or length                             |
| `ok`                   | Boolean from map/type assertion             |
| `err`                  | Error values                                |
| `ctx`                  | context.Context                             |
| `b`                    | bytes or buffer                             |
| `r`, `w`               | io.Reader, io.Writer                        |
| `t`                    | \*testing.T                                 |
| `s`                    | String (when scope is tiny)                 |
| `idx`                  | Index (when `i` is ambiguous)               |
| `pfx`, `sfx`           | Prefix, suffix                              |
| `cfg`                  | Config struct                               |
| `opts`                 | Options struct                              |
| `pl`                   | Execution plan                              |
| `sid`, `fid`, `nid`    | Step, flow, and node IDs                    |
| `cat`                  | Catalog state in local scope                |
| `flow`, `step`, `work` | Current flow/step/work value in local scope |
| `h`                    | `api.HealthState` or other health value in tight scope |

Prefer the established short forms already used in this codebase when the type is obvious from context, especially in tests:

| Name   | Usage                                          |
| ------ | ---------------------------------------------- |
| `sp`   | `api.Space` and space-like locals              |
| `st`   | `*api.Step` and step-like locals               |
| `fl`   | `api.FlowState` and flow-like locals           |
| `ex`   | `api.ExecutionState` and execution-like locals |
| `pl`   | `*api.ExecutionPlan`                           |
| `sid`  | `api.StepID`                                   |
| `fid`  | `api.FlowID`                                   |
| `nid`  | `api.NodeID`                                   |
| `h`    | `api.HealthState`                              |
| `cat`  | `api.CatalogState`                             |

Examples:

```go
st := helpers.NewSimpleStep("test-step")
sp := api.Space{ID: "payments", Name: "Payments"}
pl := &api.ExecutionPlan{Goals: []api.StepID{st.ID}, Steps: api.Steps{st.ID: st}}
fl := env.WaitForFlowStatus("wf-test", func() {
    err := env.Engine.StartFlow("wf-test", pl)
    assert.NoError(t, err)
})
ex := fl.Executions[st.ID]
sid := api.StepID("step-a")
fid := api.FlowID("wf-test")
nid := api.NodeID("node-a")
h := api.HealthState{Status: api.HealthHealthy}
```

Use `sid`, `fid`, and `nid` instead of longer local names like `stepID`, `flowID`, and `nodeID` when the scope is local and the meaning is already clear.
Longer forms such as `stepID`, `flowID`, and `nodeID` are still acceptable, and often preferable, for function parameters and other API boundaries where explicitness matters more than brevity.
Similarly, prefer `h` for local `api.HealthState` values, but `health` is acceptable, and often preferable, for function parameters and other API boundaries.

### Argument and Result Structs

Structs with an `Args` suffix are parameter bundles for a single function. Structs with a `Res` suffix are result bundles returned by a single function. Both have strict rules:

- **Threshold (hard rule)**: a function taking **5 or more arguments** must bundle them into an `Args` struct; a function returning **3 or more results** must bundle them into a `Res` struct. Below those counts, pass/return values directly.
- **Success indicators are exempt from the result count**: a trailing `bool` (the `ok` idiom) or `error` signals success/failure, not data, so it never counts toward the 3-result threshold. `(resolveStepRes, bool)` and `(api.FlowState, api.CatalogState, error)` are fine — count only the data values. The correct shape is a `Res` struct **plus** the bool: `func resolveStep(...) (resolveStepRes, bool)`, not an `ok` field stuffed inside the struct.
- **Same-type adjacency**: the hazard is exactly *two values of the same type next to each other* — nothing else. Distinct types self-disambiguate: `(int, bool)` is fine, `(*api.Step, int)` is fine, because the type tells you which value is which. Two of the same type do not: `(bool, bool)` — which bool is which? `(api.StepID, api.StepID)` — source then target, or target then source? That order is a convention, not a fact the types enforce, so a swap compiles silently — the same hazard as positional struct literals. When two adjacent params or results share a type, give them names: an `Args`/`Res` struct with named fields, or distinct named types (`type FlowID string`), so the meaning lives in the code, not in an assumed convention.
- **Lifetime**: an `Args` struct must not outlive the call site where it is passed; a `Res` struct must not outlive the call site where it is received. Neither may be stored, forwarded to another function, or returned further up the call chain.
- **If a struct crosses more than one call site** it is not an Args/Res struct — rename it to a plain descriptive name (no suffix) and use pointer currency (`*T`) when passing it.
- **Placement**: each Args/Res struct must be declared immediately before the function that accepts or returns it. If one function has both, declare them together in a single `type (...)` block immediately before that function. Never group them with unrelated types at the top of the file.
- **Value passing**: at their single call site, Args/Res structs are passed and returned by value (no `*`), whatever their size. They are short-lived and stack-allocated by design.
- **Pointer threshold (plain-named structs)**: a struct that crosses call sites is passed by value or by pointer according to its size, not by habit. Up to **32 bytes** (4 machine words) always by value. **33–64 bytes**: by value, unless it is read by three or more functions in a chain. Over **64 bytes** (one cache line): by pointer. Regardless of size, use a pointer when the callee mutates the struct or its identity matters. A `&x` that never escapes stays on the stack: the cost of a pointer there is the indirection, not an allocation.
- **Field names must stand alone**: a name that worked as a positional parameter (short, disambiguated by position and the surrounding call) does not automatically work as a named struct field read on its own at a call site. Spell out abbreviations that aren't immediately decodable without reading the function body: `st` → `step`, `pl` → `plan`, `h` → `health`, `e` → `engine`. Two fields of the same type still need names that disambiguate them beyond position — `parent api.FlowID` next to `id api.FlowID` is the same-type-adjacency hazard above; use `parentFlowID` / `childFlowID`.
- **Literal formatting**: if a struct literal fits on one line, leave it on one line. If it wraps, use exactly one field per line — never pack two or more fields onto a wrapped line.

```go
// Good — declared immediately before its function, used at one call site
type startCompensatingServerArgs struct {
    engineURL  string
    stepName   api.Name
    stepID     api.StepID
    handle     builder.StepHandler
    compensate builder.CompensateHandler
}

func startCompensatingServer(
    t *testing.T, args startCompensatingServerArgs,
) string { ... }

// Good — fits on one line, stays on one line
startFlow(startFlowArgs{fid: fid, plan: pl, state: st})

// Good — wraps, so one field per line
startFlow(startFlowArgs{
    fid:      fid,
    plan:     pl,
    state:    st,
    deadline: time.Now().Add(DefaultTimeout),
    parent:   parentFlowID,
})

// Bad — wrapped but multiple fields share a line
startFlow(startFlowArgs{
    fid: fid, plan: pl,
    state: st, deadline: time.Now().Add(DefaultTimeout),
})

// Bad — declared in a top-level type block far from its function
type (
    startFlowArgs struct { ... }  // ← wrong place
    someOtherType struct { ... }
)

// Bad — forwarded to a second function (no longer a single call site)
func outer(args startFlowArgs) {
    args.fid = ""      // mutates
    inner(args)        // forwarded — rename and use *startFlowParams
}
```

Keep `*testing.T` as the first positional parameter of test helpers, outside the argument struct even though it counts toward the threshold, so test-related functions stay identifiable at a glance.

Exported APIs may instead use descriptive request, response, or state type names without `Args` or `Res`. Declare those exported types at the top of the file with the other exported types; pointers are acceptable when appropriate.

### Function Signature Wrapping

When a function signature is too long for one line, keep as many parameters as fit on the first line and wrap the remainder on the next line(s). Go one parameter per line only when a line would still exceed the limit.

Example with more parameters:

```go
func WaitForStepEvents(
	t *testing.T, consumer topic.Consumer[*timebox.Event], flowID api.FlowID,
	stepID api.StepID, count int, timeout time.Duration,
	eventTypes ...api.EventType,
) {
```

### Function Names

Verb + noun. Get/Set only when accessing fields:

```go
// Good
func (e *Engine) ProcessEvent(event *Event)
func (s *Store) LoadFlow(id FlowID) (*FlowState, error)
func (s *Store) SaveFlow(flow *FlowState) error
func (c *Client) FetchStep(id StepID) (*Step, error)

// Bad - Get/Set for non-field access
func (s *Store) GetFlowFromDatabase(id FlowID)  // Use Load
func (c *Client) GetStepFromAPI(id StepID)      // Use Fetch
```

### Constructor Names

`New` prefix, return pointer:

```go
// Good
func NewEngine(store Store) *Engine
func NewScheduler(ctx context.Context, interval time.Duration) (*Scheduler, error)

// Bad
func CreateEngine(store Store) *Engine
func MakeEngine(store Store) *Engine
```

### Interface Names

Single-method interfaces use `-er` suffix. Capabilities, not implementations:

```go
// Good - describes what it does
type EventConsumer interface {
    Consume() (*Event, error)
}

type StepInvoker interface {
    Invoke(ctx context.Context, req *Request) (*Response, error)
}

// Bad - describes what it is
type EventConsumerInterface interface { ... }
type IEventConsumer interface { ... }
```

### Constant Names

`Default` prefix for defaults. `Max`/`Min` for limits:

```go
// Good
const (
    DefaultTimeout = 30 * time.Second
    DefaultRetries = 10
    MaxConcurrency = 100
    MinBackoff     = 100
)

// Bad - unclear what 30 means
const Timeout = 30 * time.Second
```

### Error Names

`Err` prefix, grouped in `var` block:

```go
var (
    ErrNotFound     = errors.New("not found")
    ErrInvalidState = errors.New("invalid state")
    ErrTimeout      = errors.New("operation timed out")
)
```

### Boolean Names

Name a boolean for the state it holds; Go reads it as a predicate already:

```go
// Good
if active { ... }
if flow.Terminal { ... }
if hasActiveWork(flow) { ... }  // Functions can use has/is

// Acceptable in struct fields when clarity needed
type Config struct {
    Enabled bool
    Ready   bool
}

// Bad - redundant prefix
if isActive { ... }
if flow.IsTerminal { ... }
```

### Acronyms

All caps for acronyms, even in camelCase:

```go
// Good
type HTTPClient struct {}
func (c *Client) GetURL() string
type FlowID string
var xmlParser Parser

// Bad
type HttpClient struct {}
func (c *Client) GetUrl() string
type FlowId string
```

## Formatting

### Line Width

Maximum 80 characters per line (tabs count as 4 spaces). This applies to code _and_ comments. Keep short argument lists on a single line when they fit; only break lines when the 80-character limit would be exceeded. When wrapping function signatures or call arguments, pack as many arguments per line as will fit under the limit before wrapping again. When you must wrap, break after the opening paren:

```go
func NewScheduler(
	ctx context.Context, interval time.Duration, handler WorkHandler,
) (*Scheduler, error) {
```

```go
c, err := client.NewClient("embedded://", client.WithEmbedded(tr))
```

### Multi-line Calls with \*testing.T

When a call wraps and the test instance (`t`) is its receiver or its first argument, keep `t` on the first line, break immediately after it, and close on the last argument:

```go
WaitForFlowEvents(t,
	consumer, flowIDs, timeout, api.EventTypeFlowStarted)
```

```go
assert.Equal(t,
	api.FlowID("parent-flow"), metaFlowID(childState.Metadata))
```

```go
t.Fatalf("unexpected required match location: %s",
	got.RequiredMatch.Location)
```

Every other wrapped call breaks after the opening paren, with a trailing comma and the closing paren on its own line:

```go
err := env.Engine.RegisterStep(
	helpers.NewSimpleStep("count-step-1"),
)
```

## File Organization

### Imports

Run `goimports` on all files. It handles grouping and sorting automatically.

### No Function-Scoped Type or Const Declarations

All `type` and `const` declarations live at package level, in the appropriate block alongside the rest of the package's types and constants.

```go
// Bad — type declared inside a function
func process() {
    type work struct{ id api.StepID }  // FORBIDDEN
    const limit = 100                  // FORBIDDEN
}

// Good — all at package level
type work struct{ id api.StepID }

const limit = 100

func process() { ... }
```

This applies to test files as well.

### Top-Level Declaration Order

1. `type` declarations (use a block only when declaring multiple types). Ordering rule: exported types come before unexported types; within the same visibility, if a type uses another type, the using type goes first.
2. `const` declarations (use a block only when declaring multiple constants)
3. `var` declarations (use a block only when declaring multiple vars; exception: errors always use a `var` block)
4. Exported functions (including constructors like `New...`)
5. Exported methods
6. Unexported methods
7. Unexported helper functions

```go
package engine

type (
	Engine struct { ... }
	EventConsumer = topic.Consumer[*timebox.Event]
)

const DefaultTimeout = 30 * time.Second

var (
	ErrNotFound = errors.New("not found")
	ErrExists   = errors.New("already exists")
)

func New(...) *Engine { ... }

func (e *Engine) Start() { ... }           // exported
func (e *Engine) Stop() error { ... }      // exported

func (e *Engine) processEvent(...) { ... } // unexported
func helperFunc(...) { ... }               // unexported helper
```

### Method Ordering

1. Constructor (`New...`)
2. Exported methods grouped by functionality
3. Unexported methods that support the exported ones
4. Pure helper functions (non-methods) at the bottom

Related methods stay together. Within each group, order by call chain or first use. Unexported helpers appear after the exported methods that use them.

`func main()` is the exception to all ordering rules above: it is the
package's entry point, so it always comes first among functions, immediately
after the top-level `type`/`const`/`var` declarations — before constructors
and any other helper it calls.

### Concern Grouping

Within a package, organize files around real concerns, not arbitrary helper categories. For the engine runtime, prefer lifecycle or stage-oriented grouping when that matches the code's behavior:

- `engine-start.go`, `engine-stop.go`
- `flow-start.go`, `flow-continue.go`, `flow-stop.go`
- `step-start.go`, `step-continue.go`, `step-stop.go`
- `work-start.go`, `work-continue.go`, `work-stop.go`

Let callers import the owning package and use its calls and errors directly.

## Struct Literals

**NEVER construct a struct using positional field order.** Always use named fields. Positional literals are fragile: a field reorder or insertion silently compiles and corrupts data.

```go
// Good
api.HealthState{Status: api.HealthHealthy, Since: now, Reason: reason}

// Bad — positional, breaks silently on field reorder
api.HealthState{api.HealthHealthy, now, reason}
```

The only exception is single-field structs where the field name adds no information.

## Control Flow

### Early Returns

Use guard clauses to reject invalid preconditions before substantial main logic. No else when early return works:

```go
// Good
func processStep(step *StepInfo) error {
	if step == nil {
		return ErrNilStep
	}
	if !step.IsValid() {
		return ErrInvalid
	}
	// main logic
	return nil
}

// Bad
func processStep(step *StepInfo) error {
	if step != nil {
		if step.IsValid() {
			// main logic
			return nil
		} else {
			return ErrInvalid
		}
	} else {
		return ErrNilStep
	}
}
```

Guard clauses earn their place in branching functions. When a value and success indicator are used only by a short success path, declare them in the `if` initializer, keep the value scoped to that branch, and put the fallback afterward. Use this form when the condition is only the success indicator, or the success indicator plus one simple boolean, nil, equality, or predicate check. Once the condition needs more than two checks, becomes multi-line, or the success branch grows substantial logic, switch to guard clauses.

```go
// Good
func lookupWork(token api.Token) (api.WorkItem, bool) {
	if work, ok := exec.WorkItems[token]; ok && work.Active() {
		return work, true
	}
	return api.WorkItem{}, false
}

// Bad — value and ok escape the only branch that uses them
func lookupWork(token api.Token) (api.WorkItem, bool) {
	work, ok := exec.WorkItems[token]
	if !ok || !work.Active() {
		return api.WorkItem{}, false
	}
	return work, true
}
```

### Multi-Assignment

Give each independent value its own statement, so the code reads top to bottom instead of pairing names on the left with values on the right. One statement carries multiple variables when it routes a single call's return values, where the signature fixes the pairing.

```go
// Good — routing one call's multi-return
flow, err := eng.GetFlowState(fid)

// Good — independent sources, one per line
cat := st.Catalog
cluster := st.Cluster

// Bad — independent sources crammed into one statement
cat, cluster := st.Catalog, st.Cluster
```

This applies to plain assignment (`=`) as well as `:=`.

### Nesting Limit

Maximum one level of conditional nesting. Exception: when early return would cause code duplication.

```go
// Acceptable nesting to avoid duplicating the publish call
func updateHealth(stepID api.ID, health Health) error {
	if stepState, ok := state.Steps[stepID]; ok {
		if stepState.Health == health {
			return nil
		}
	}
	return ds.publish(ctx, events.HealthChanged, data)
}
```

## Testing

### Coverage Target

Minimum 90% test coverage.

### Black-Box Testing Only

All tests use `package_test` suffix:

```go
package engine_test  // Good
package engine       // Bad
```

### Test Naming

Function names short, subtests can be longer:

```go
// Good - short function name
func TestScheduler(t *testing.T) {
    t.Run("returns error when store unavailable", func(t *testing.T) {
        // ...
    })
    t.Run("executes handler on interval", func(t *testing.T) {
        // ...
    })
}

// Bad - underscores are extraneous
func TestStore_Get(t *testing.T) { ... }
func TestEngine_Start(t *testing.T) { ... }

// Bad - function name is a novel
func TestSchedulerReturnsErrorWhenStoreIsUnavailable(t *testing.T) { ... }
func TestEngineShouldStartCorrectlyWhenConfigIsValid(t *testing.T) { ... }
```

### Assertions

Use `testify/assert`, with the assertion alone and no message args:

```go
// Good
assert.NoError(t, err)
assert.Equal(t, expected, actual)
assert.True(t, ok)

// Bad - require stops test early
require.NoError(t, err)

// Bad - no message arguments
assert.NoError(t, err, "should not error")
assert.Equal(t, expected, actual, "values should match")
```

### Test Organization

- Table-driven tests for multiple scenarios
- Subtest descriptions with `t.Run()`
- `t.Helper()` in test utilities
- Keep test files aligned with source concerns when the split is clear

If the runtime code is grouped by stage or lifecycle, the tests should mirror that grouping:

- `flow-start_test.go`
- `flow-continue_test.go`
- `flow-stop_test.go`
- `step-start_test.go`
- `step-continue_test.go`
- `step-stop_test.go`
- `work-start_test.go`
- `work-continue_test.go`
- `work-stop_test.go`

Once the source splits cleanly, split its tests to match.

## Comments

Every comment earns its place. Code that needs prose to be understood is code that needs rewriting.

### Godoc

**Exported** funcs, methods, types, consts, and vars carry godoc, capped at 3 lines. Say what it does. Needing more than 3 lines means the thing itself wants simplifying:

```go
// Scheduler manages periodic work execution using timebox.Store,
// supporting configurable intervals and retry behavior
type Scheduler struct {
```

Sentinel error vars are the exception: the message is the documentation.

**Unexported** funcs and methods stand on their own name and signature. Add a godoc, capped at 2 lines, when the behavior is genuinely non-trivial:

```go
// unexported and self-evident, no comment
func clampRetry(d time.Duration, max int) time.Duration {

// the "why" is invisible from the signature, 2 lines max
// diffDebounce: single async debounce; the gutter trails a keystroke
const diffDebounce = 50 * time.Millisecond
```

End the last sentence of a godoc without a period.

### Comment Density

Match the surrounding code. A field whose siblings carry no doc comment gets none either.

### Inline Comments

An inline comment explains WHY, capped at 2 lines. Code says what it does on its own:

```go
// Bad, restates the code
bucket, err := blob.OpenBucket(ctx, url)  // Open the bucket
return err                                 // Return the error

// Good, explains WHY, 2 lines max
// Missing key is not an error; deletion is idempotent by design
if gcerrors.Code(err) == gcerrors.NotFound {
	return nil
}
```

Explaining the mechanism the reader can already see, restating a name, or justifying a decision belongs in the commit message, not the source.

## Global State

**Mutable package-level variables are absolutely forbidden.** This includes counters, caches, registries, or any other state that can be mutated after initialization.

```go
// Bad — mutable global state
var idCounter atomic.Int64

// Good — state lives on the owning struct
type Engine struct {
    nextID int
}
```

Package-level `var` declarations are permitted only for:

- Sentinel error values (`var ErrNotFound = errors.New(...)`)
- Compile-time interface assertions (`var _ Foo = (*Bar)(nil)`)
- Truly immutable lookup tables that are never reassigned (treat them as constants; document if a slice element could be mutated)

## No Cross-Package Var Aliasing

**Never declare `var Foo = otherpkg.Foo` to re-export another package's identifier under a local name.** If a package needs a value another package already owns, import that package and reference the value directly.

```go
// Bad — re-exports api's sentinel under a local name
var ErrStepNotFound = api.ErrStepNotFound

// Good — call sites use the owning package's identifier
return api.ErrStepNotFound
```

This applies to sentinel errors, constants, and any other exported value.

## Interface Compliance

Compile-time interface checks:

```go
var _ StepInvoker = (*Client)(nil)
```

## Error Handling

- **Never panic** - always return errors
- **Typed errors only** - All production code must use package-level vars with `Err` prefix
- **Pattern: `%w: context`** — wrapped error first, then context variable
- Plain error messages acceptable only in examples/documentation
- Handle errors immediately, early return

**Production Code - Always Use Typed Errors:**

```go
var (
	ErrStepNotInPlan        = errors.New("step not in execution plan")
	ErrWorkItemNotFound     = errors.New("work item not found")
	ErrInvalidWorkTransition = errors.New("invalid work state transition")
)

// Good - %w: %s pattern with typed error
if x == nil {
    return fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
}

// Good - typed error with multiple context values
if !workTransitions.CanTransition(work.Status, toStatus) {
    return fmt.Errorf("%w: %s -> %s", ErrInvalidWorkTransition,
        work.Status, toStatus)
}

// Good - return typed error directly
if x == nil {
    return nil, ErrNotFound
}

// Bad - plain message in production code (no typed error)
if x == nil {
    return fmt.Errorf("work item not found: %s", token)  // NO! Use typed error
}

// Bad - context before wrapped error
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to process: %w", err)  // Wrong order
}

// Bad - never panic
if x == nil {
    panic("x is nil")  // NO!
}
```

**Testing - Use errors.Is() to Check Typed Errors:**

Tests should use `errors.Is()` to check for specific error types, not `strings.Contains()`:

```go
// Good - use errors.Is for typed errors
err := tx.checkWorkTransition(stepID, token, toStatus)
assert.True(t, errors.Is(err, ErrWorkItemNotFound))

// Bad - fragile string matching
assert.True(t, strings.Contains(err.Error(), "work item not found"))
```

This enables robust error checking without brittle string comparisons. Typed errors are also easier to handle programmatically.

**Examples/Documentation Only - Plain Messages OK:**

```go
// Only acceptable in README examples, not in engine code
return fmt.Errorf("invalid configuration: %s", reason)
```

## Constants

- No magic numbers
- Group related constants
- Use typed constants when meaningful

```go
const (
	DefaultTimeout = 30 * time.Second
	DefaultRetries = 10
	DefaultBackoff = 1000
)
```
