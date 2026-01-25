# Go Style Guide

## Naming Conventions

### Receiver Names

Single lowercase letter, first letter of type name:

```go
// Good
func (e *Engine) Start() {}
func (a *flowActor) process() {}
func (s *Store) Get() {}
func (c *Client) Do() {}

// Bad
func (engine *Engine) Start() {}
func (self *Engine) Start() {}
func (this *Engine) Start() {}
```

### Variable Names

**Prefer short names.** The closer a variable is used to where it's declared, the
shorter it can be. Loop variables can be single letters.

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

**Longer names for wider scope** (exported functions, struct fields):

```go
// Good - clear at API boundaries
func (e *Engine) StartFlow(
    flowID api.FlowID, goalSteps []api.StepID, initState api.Args,
) (*api.FlowState, error)

// Good - descriptive struct fields
type FlowState struct {
    FlowID     api.FlowID
    Status     FlowStatus
    Executions map[StepID]*Execution
}
```

**Idiomatic short names**:

| Name | Usage |
|------|-------|
| `i`, `j`, `k` | Loop indices |
| `n` | Count or length |
| `ok` | Boolean from map/type assertion |
| `err` | Error values |
| `ctx` | context.Context |
| `b` | bytes or buffer |
| `r`, `w` | io.Reader, io.Writer |
| `t` | *testing.T |
| `s` | String (when scope is tiny) |
| `idx` | Index (when `i` is ambiguous) |
| `pfx`, `sfx` | Prefix, suffix |
| `cfg` | Config struct |
| `opts` | Options struct |

### Function Signature Wrapping

When a function signature is too long for one line, keep as many parameters as fit on the first line and wrap the remainder on the next line(s). Do not put one parameter per line unless the line would still exceed the limit.
| `ev` | Event |

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
func NewArchiveWorker(ctx context.Context, url string) (*ArchiveWorker, error)

// Bad
func CreateEngine(store Store) *Engine
func MakeEngine(store Store) *Engine
```

### Interface Names

Single-method interfaces use `-er` suffix. Capabilities, not implementations:

```go
// Good - describes what it does
type Archiver interface {
    Archive(ctx context.Context, key string) error
}

type EventConsumer interface {
    Consume() (*Event, error)
}

// Bad - describes what it is
type ArchiverInterface interface { ... }
type IArchiver interface { ... }
```

### Constant Names

`Default` prefix for defaults. `Max`/`Min` for limits:

```go
// Good
const (
    DefaultTimeout   = 30 * time.Second
    DefaultRetries   = 10
    MaxConcurrency   = 100
    MinBackoffMs     = 100
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

Avoid `is`/`has` prefix (redundant in Go):

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

Maximum 80 characters per line (tabs count as 4 spaces). Keep short argument lists on a single line when they fit; only break lines when the 80-character limit would be exceeded. When you must wrap, break after the opening paren and align arguments on new lines:

```go
func NewArchiveWorker(
	ctx context.Context, bucketURL, prefix string,
) (*ArchiveWorker, error) {
```

```go
c, err := client.NewClient("embedded://", client.WithEmbedded(tr))
```

### Multi-line Calls with *testing.T

When a function call wraps and the first argument is the test instance (`t`), keep `t` on the first line and break immediately after it. Do not place `t` alone on the next line.

```go
WaitForFlowEvents(t,
	consumer, flowIDs, timeout, api.EventTypeFlowStarted,
)
```

```go
assert.Equal(t,
	api.FlowID("parent-flow"), metaFlowID(childState.Metadata),
)
```

## File Organization

### Imports

Run `goimports` on all files. It handles grouping and sorting automatically.

### Top-Level Declaration Order

1. `type` declarations (use a block only when declaring multiple types). Ordering rule: if a type uses another type, the using type goes first.
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

Related methods stay together. Within each group, order by call chain or
first use. Unexported helpers appear after the exported methods that use
them.


## Control Flow

### Early Returns

Use guard clauses to minimize nesting. No else when early return works:

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

### Nesting Limit

Maximum one level of conditional nesting. Exception: when early return
would cause code duplication.

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
package archive_test  // Good
package archive       // Bad
```

### Test Naming

Function names short, subtests can be longer:

```go
// Good - short function name
func TestArchive(t *testing.T) {
    t.Run("returns error when bucket unavailable", func(t *testing.T) {
        // ...
    })
    t.Run("deletes key after upload", func(t *testing.T) {
        // ...
    })
}

func TestStore_Get(t *testing.T) { ... }
func TestEngine_Start(t *testing.T) { ... }

// Bad - function name is a novel
func TestArchiveReturnsErrorWhenBucketIsUnavailable(t *testing.T) { ... }
func TestEngineShouldStartCorrectlyWhenConfigIsValid(t *testing.T) { ... }
```

### Assertions

Use `testify/assert` only. Never `testify/require`. Never include message args:

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

## Comments

### Godoc

Exported symbols need godoc that adds value beyond the name:

```go
// ArchiveWorker implements flow archiving policy using timebox.Store,
// supporting external consumers for long-term storage
type ArchiveWorker struct {
```

Skip godoc when the name is self-documenting:

```go
func NewArchiveWorker(...) (*ArchiveWorker, error) {
```

Godoc rule: the last sentence of a comment should not end with a period.

### Inline Comments

Never add self-evident comments. Only comment non-obvious logic:

```go
// Bad
bucket, err := blob.OpenBucket(ctx, url)  // Open the bucket
return err                                 // Return the error

// Good - explains WHY
// Delete succeeds on missing key to make archiving idempotent
if gcerrors.Code(err) == gcerrors.NotFound {
	return nil
}
```

## Interface Compliance

Compile-time interface checks:

```go
var _ Archiver = (*ArchiveWorker)(nil)
```

## Error Handling

- **Never panic** - always return errors
- Typed errors as package-level vars with `Err` prefix
- Wrap with context: `fmt.Errorf("context: %w", err)`
- Handle errors immediately, early return

```go
var (
	ErrNotFound = errors.New("not found")
	ErrExists   = errors.New("already exists")
)

// Good - return error
if x == nil {
    return nil, ErrNotFound
}

// Bad - never panic
if x == nil {
    panic("x is nil")  // NO!
}
```

## Constants

- No magic numbers
- Group related constants
- Use typed constants when meaningful

```go
const (
	DefaultTimeout     = 30 * time.Second
	DefaultRetries     = 10
	DefaultBackoffMs   = 1000
)
```
