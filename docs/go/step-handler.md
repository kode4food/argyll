# Step Handler (StepContext)

## Basic Usage

```go
func handleStep(sctx *builder.StepContext) (api.StepResult, error) {
    userID := sctx.GetString("user_id")

    sctx.Logger().Info("processing", "user_id", userID)

    result := processUser(userID)

    return *api.NewResult().WithOutput("result", result), nil
}
```

## Identity

```go
flowID := sctx.FlowID()       // Current workflow ID
stepID := sctx.StepID()       // Current step ID
token := sctx.WorkToken()     // Unique work item token
ctx := sctx.Context()         // Underlying context.Context
```

## Arguments

### Type-Safe Getters

```go
name := sctx.GetString("name")   // Returns "" if not found
count := sctx.GetInt("count")    // Returns 0 if not found
price := sctx.GetFloat("price")  // Returns 0.0 if not found
active := sctx.GetBool("active") // Returns false if not found

// Generic accessor with existence check
if value, ok := sctx.Get("optional"); ok {
    // Handle value
}

// Access all arguments
allArgs := sctx.Args()
```

Type conversions:
- `GetInt()` handles int, float64, int64
- `GetFloat()` handles float64, int, int64

## Workflow State

### Inspection

```go
// Get workflow state
state, err := sctx.GetWorkflowState()
if err != nil {
    return api.StepResult{}, err
}

// Check specific step execution
exec, err := sctx.GetStepExecution("previous-step")
if err != nil {
    return api.StepResult{}, err
}

if exec.Status == api.StepCompleted {
    sctx.Logger().Info("previous step completed")
}

// Read workflow attributes
if userID, ok := sctx.GetAttribute("user_id"); ok {
    sctx.Logger().Info("found user_id", "value", userID)
}
```

### Mutation

```go
// Update single attribute
err := sctx.SetAttribute("processed_count", 42)

// Update multiple attributes
err = sctx.UpdateAttributes(map[api.Name]any{
    "status": "processing",
    "timestamp": time.Now().Unix(),
    "items_processed": 100,
})
```

## Logging

The logger is pre-configured with flow_id and step_id:

```go
sctx.Logger().Info("starting processing")

sctx.Logger().Info("user resolved",
    "user_id", userID,
    "email", email,
    "status", "active")

sctx.Logger().Debug("detailed debug info")
sctx.Logger().Warn("potential issue")
sctx.Logger().Error("failed to process", "error", err)
```

## Context for Cancellation

```go
ctx := sctx.Context()

// HTTP requests
req, err := http.NewRequestWithContext(ctx, "GET", url, nil)

// Database queries
rows, err := db.QueryContext(ctx, query)

// Check for cancellation
select {
case <-ctx.Done():
    return api.StepResult{}, ctx.Err()
default:
    // Continue
}
```

## Complete Example

```go
func handleOrderProcessor(sctx *builder.StepContext) (api.StepResult, error) {
    orderID := sctx.GetString("order_id")
    items := sctx.Get("items")
    priority := sctx.GetString("priority")

    sctx.Logger().Info("processing order",
        "order_id", orderID,
        "priority", priority)

    // Check if already processed
    if processed, ok := sctx.GetAttribute("order_" + api.Name(orderID)); ok {
        if processed.(bool) {
            sctx.Logger().Warn("already processed", "order_id", orderID)
            return *api.NewResult().WithOutput("processed", false), nil
        }
    }

    // Process
    totalAmount := calculateTotal(items)

    // Update workflow state
    err := sctx.UpdateAttributes(map[api.Name]any{
        "order_" + api.Name(orderID): true,
        "last_processed_at": time.Now().Unix(),
    })
    if err != nil {
        sctx.Logger().Error("failed to update state", "error", err)
        return api.StepResult{}, err
    }

    sctx.Logger().Info("order processed",
        "order_id", orderID,
        "amount", totalAmount)

    return *api.NewResult().
        WithOutput("processed", true).
        WithOutput("total_amount", totalAmount), nil
}
```

## Advanced

### Raw Metadata

```go
metadata := sctx.Metadata()

if customField, ok := metadata["custom_field"]; ok {
    // Use custom field
}
```
