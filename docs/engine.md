# Argyll Engine: Architecture and Execution Model

This document explains how the Argyll engine works internally, covering the core mechanisms for step registration, script compilation, execution planning, and flow orchestration.

---

## Table of Contents

1. [Step Registration and Updates](#step-registration-and-updates)
2. [Predicate and Script Compilation](#predicate-and-script-compilation)
3. [Plan Generation and Preview](#plan-generation-and-preview)
4. [Flow Execution](#flow-execution)

---

## Step Registration and Updates

### What is a Step?

A step is a unit of computation in Argyll. Think of it as a single task that can be performed, like "send email", "query database", or "calculate result". Each step has:

- **An identity**: A unique ID and human-readable name
- **A type**: Either a synchronous HTTP call, asynchronous HTTP call, script, or flow
- **Inputs and outputs**: Declared as attributes with specific types and roles
- **Optional execution logic**: Either an HTTP endpoint to call or a script to run
- **Optional conditional logic**: A predicate that determines if the step should run
- **Execution configuration**: Retry policies and parallelism settings

### The Four Types of Steps

**Synchronous HTTP Steps** call an HTTP endpoint and wait for the response before continuing. The endpoint returns the step's outputs directly in the HTTP response.

**Asynchronous HTTP Steps** also call an HTTP endpoint, but they don't wait for outputs. Instead, the endpoint receives a webhook URL that it calls later when the work is done. This allows long-running operations to complete without tying up resources.

**Script Steps** execute embedded code (written in Lua or Ale) locally within the engine. The script receives inputs as arguments and returns outputs as its result.

**Flow Steps** start a child flow using a list of goal steps and optional input/output mappings. Inputs are mapped into the child flow before execution, and selected outputs can be mapped back into the parent flow when the child completes.

### How Registration Works

When you register a step, the engine performs a series of validations before accepting it. This happens in a specific order to ensure data integrity.

#### Pre-Transaction Validation

First, before any state changes are made, the engine validates all scripts:

If the step is a script step, the engine compiles the main script to check for syntax errors. If the step has a predicate (conditional logic), that script is also compiled and checked. This happens early because there's no point in storing a step definition if its code won't even compile.

The compilation process is language-specific. For Lua scripts, the engine wraps the code with argument binding logic, then compiles it to bytecode to verify syntax. For Ale scripts, the engine wraps the expression in a lambda function and evaluates it to ensure it's valid.

If any script fails to compile, registration is rejected immediately without making any changes to the engine state.

#### Transaction-Based Validation

Once scripts pass validation, the engine enters a transaction to validate against the current state:

**Duplicate Detection**: The engine checks if a step with the same ID already exists. If it does, registration fails unless the new step is identical to the existing one (making registration idempotent).

**Type Consistency**: The engine enforces that attribute types are consistent across all steps. If step A declares an attribute "user" as a string, then step B cannot declare "user" as a number. This prevents type conflicts during execution.

Here's how this works: The engine collects all attribute names and their types from every registered step. For each attribute in the new step, it checks if any other step has defined that attribute with a different type. If there's a mismatch, registration fails with a type conflict error.

**Circular Dependency Detection**: The engine uses an attribute dependency map showing which steps provide (produce) and consume each attribute. This uses the same Provider→Attribute→Consumer structure used throughout the engine.

The algorithm works like this: Starting from the new step being registered, it recursively follows the dependency chain by looking at which steps provide its required inputs. If it encounters the original step again during traversal, that means there's a cycle. For example, if step A depends on step B, and step B depends on step C, and step C depends on step A, that's a circular dependency and registration fails.

For cycle detection, the dependency map is computed from all registered steps plus the new step being validated. It maps each attribute name to a Dependencies struct containing the list of provider steps (which produce that attribute) and consumer steps (which require it as input). This structure is identical to the one maintained in EngineState for execution planning, ensuring consistency across validation and planning.

#### Event Emission

If all validations pass, the engine raises a "StepRegisteredEvent" within the transaction. This event is the single source of truth for the state change. The event contains the complete step definition.

When the transaction commits, the event is persisted to the event log. An event applier function then processes the event and updates the engine state by:
- Adding the step to the steps registry
- Rebuilding the attribute dependency graph (cached in EngineState.Attributes)
- Initializing its health status as "unknown"
- Updating the last-modified timestamp

#### Post-Transaction Updates

After the transaction commits successfully, the engine performs one final action: if the step contains any scripts (main script or predicate), it updates the health status to "healthy". This indicates that the step's code has been validated and is ready for use.

### How Updates Work

Updating a step follows a similar process but with inverted validation logic:

The step **must** already exist in the registry. If you try to update a step that hasn't been registered, the update fails.

The same script validation, type consistency checking, and circular dependency detection happens as with registration.

If the new step definition is identical to the existing one, the update is a no-op (idempotent behavior).

If validations pass and the definition has changed, a "StepUpdatedEvent" is raised, persisted, and applied to update the engine state with the new definition and rebuild the cached dependency graph.

### Event Sourcing Pattern

The engine uses event sourcing, which means all state changes are recorded as immutable events rather than directly mutating state. This provides several benefits:

**Auditability**: You can see the complete history of every step that was registered or updated.

**Recovery**: If the engine crashes, it can rebuild its state by replaying all events from the log.

**Consistency**: All state changes go through the same event application logic, preventing inconsistencies.

#### Events Emitted During Step Registration

**StepRegisteredEvent** - Emitted when a new step is successfully registered:
```json
{
  "type": "step_registered",
  "timestamp": "2025-12-01T10:30:00Z",
  "data": {
    "step": {
      "id": "send-email",
      "name": "Send Email",
      "type": "async",
      "http": {
        "url": "https://api.example.com/send-email",
        "method": "POST"
      },
      "attributes": {
        "recipient": {
          "role": "required",
          "type": "string"
        },
        "subject": {
          "role": "required",
          "type": "string"
        },
        "body": {
          "role": "optional",
          "type": "string",
          "default": "\"Hello!\""
        },
        "message_id": {
          "role": "output",
          "type": "string"
        }
      }
    }
  }
}
```

**StepUpdatedEvent** - Emitted when an existing step's definition changes:
```json
{
  "type": "step_updated",
  "timestamp": "2025-12-01T11:15:00Z",
  "data": {
    "step": {
      "id": "send-email",
      "attributes": {
        "sent_at": {"role": "output", "type": "string"}
      }
    }
  }
}
```

**StepHealthChangedEvent** - Emitted when a step's health status changes:
```json
{
  "type": "step_health_changed",
  "timestamp": "2025-12-01T11:20:00Z",
  "data": {
    "step_id": "send-email",
    "status": "healthy"
  }
}
```

When a step becomes unhealthy:
```json
{
  "type": "step_health_changed",
  "timestamp": "2025-12-01T12:00:00Z",
  "data": {
    "step_id": "send-email",
    "status": "unhealthy",
    "error": "connection refused: https://api.example.com/send-email"
  }
}
```

### Health Status Tracking

Each step has a health status that indicates whether it's operational:

**Unknown**: Default state for steps without scripts, or before health checks complete.

**Healthy**: The step's scripts compiled successfully, or (for HTTP steps) health checks are passing.

**Unhealthy**: Script compilation failed, or (for HTTP steps) the endpoint is unreachable.

Health status changes are tracked through "StepHealthChangedEvent" events. The engine only emits these events when the status actually changes, making them idempotent.

### Attribute Specifications

Attributes are the data contract for a step. Each attribute has:

**A name**: How the attribute is identified (e.g., "user_id", "email_body")

**A type**: What kind of data it holds (string, number, boolean, object, array, or any)

**A role**: Whether it's a required input, optional input, or output

**An optional default value**: For optional inputs, the value to use if not provided

**A for-each flag**: Whether this attribute should trigger parallel work item creation

The role determines how the engine treats the attribute:

**Required inputs** must be provided either by the flow's initial state or by another step's output. If a required input can't be satisfied, the step won't execute.

**Optional inputs** can be omitted. If provided, they're used; otherwise, the default value is used (if one is specified) or the input is omitted.

**Outputs** are produced by the step and made available to downstream steps that depend on them.

### The Difference Between Registration and Update

Registration is for adding new steps to the engine. It fails if the step ID already exists (unless definitions are identical).

Update is for modifying existing steps. It fails if the step ID doesn't exist.

Both share the same validation pipeline for scripts, type consistency, and circular dependencies, but they have opposite existence checks.

---

## Predicate and Script Compilation

### What Are Predicates and Scripts?

**Scripts** are the computational logic for script-type steps. They receive inputs, perform calculations or transformations, and return outputs. Scripts are embedded directly in the step definition and executed locally by the engine.

**Predicates** are conditional expressions that determine whether a step should execute. They're evaluated before a step runs, and if the predicate returns false, the step is skipped entirely. Any step type can have a predicate.

Both scripts and predicates are written in either Lua or Ale (a Lisp-like functional language).

### Why Two Languages?

Supporting multiple languages gives users flexibility:

**Lua** is imperative and familiar to many developers. It has procedural syntax and mutable local variables, making it natural for step-by-step transformations.

**Ale** is functional and based on S-expressions. It's ideal for data transformations and expressing complex logic concisely. Being purely functional, it's also safer and easier to reason about.

Users can choose the language that best fits their use case and preferences.

### The Compilation Process

Compilation happens in multiple stages, with each language handling it differently but following the same overall pattern.

#### Stage 1: Source Wrapping

Raw scripts can't be executed directly because they need access to the step's input attributes as variables. The engine automatically wraps the user's script with boilerplate code that binds inputs to variables.

**For Lua scripts**, the engine generates local variable declarations for each input attribute. If a step has inputs "a" and "b", the wrapped script starts with:
```
local a = select(1, ...)
local b = select(2, ...)
```

This extracts the arguments passed to the Lua function and assigns them to named local variables. Then the user's script runs with access to those variables.

**For Ale scripts**, the engine wraps the expression in a lambda function. If a step has inputs "a" and "b", the wrapped script becomes:
```
(lambda (a b) <user's script>)
```

This creates a callable procedure that accepts the input arguments by name.

The order of arguments matters, so the engine always sorts attribute names alphabetically to ensure consistent ordering.

#### Stage 2: Syntax Validation and Compilation

Once wrapped, the script is compiled to check for syntax errors and to prepare it for efficient execution.

**Lua compilation** produces bytecode. The engine creates a temporary Lua state (the execution environment), loads the wrapped source code, and attempts to compile it. If compilation succeeds, the bytecode is dumped to a byte buffer and stored. If it fails, the error is returned and registration is rejected.

The bytecode is the compiled representation of the script that can be executed much faster than parsing source code each time.

**Ale compilation** evaluates the lambda expression to produce a callable procedure. The engine creates an Ale namespace (the execution environment), parses the wrapped source, and evaluates it. This produces a procedure object that can be called with arguments. If evaluation fails (syntax error or not a procedure), registration is rejected.

#### Stage 3: Caching

Compilation is expensive, so the engine caches compiled scripts to avoid recompiling them repeatedly.

The engine uses an LRU (Least Recently Used) cache with a capacity of 1024 entries per language. The cache key is the full wrapped source code, so identical scripts (even in different steps) share a single cached compilation.

When a script needs to be compiled, the engine first checks the cache. If found, it returns the cached compiled version immediately. If not found, it compiles the script, stores it in the cache, and returns it.

When the cache is full, the least recently used entry is evicted to make room for new compilations. This ensures frequently-used scripts stay in cache while one-off scripts are evicted.

The cache is thread-safe with read-write locking, allowing concurrent access from multiple flows.

### Sandboxing

Scripts run in a sandboxed environment to prevent malicious or accidental system damage.

**Lua sandboxing** excludes dangerous global functions and modules. The engine removes access to:
- File I/O operations (io library)
- Operating system functions (os library)
- Debugging capabilities (debug library)
- Package loading (require, dofile, loadfile)
- Dynamic code loading (load function)

This ensures scripts can only perform pure computation and data transformation. They cannot access the filesystem, make system calls, or load external code.

**Ale sandboxing** is inherent to the language design. Ale is purely functional with no I/O capabilities built-in, so scripts are naturally sandboxed.

### State Management for Execution

To execute scripts, the engine needs an execution environment (the runtime context where the script runs).

**Lua uses a state pool**. Creating and destroying Lua states is expensive, so the engine maintains a pool of 10 pre-created states. When a script needs to execute:

1. The engine acquires a state from the pool (blocking if all 10 are in use)
2. The sandbox is set up (dangerous functions removed)
3. The compiled bytecode is loaded
4. Input arguments are pushed onto the stack
5. The script executes
6. Results are extracted from the stack
7. The state is returned to the pool for reuse

This pooling strategy allows up to 10 scripts to execute concurrently while avoiding the overhead of state creation.

**Ale uses a shared environment**. Since Ale is purely functional with no mutable state, a single shared environment can be safely used by all scripts. There's no need for pooling or isolation.

### How Predicates Are Evaluated

Predicates determine if a step should execute. This evaluation happens just before a step is about to start, after all its required inputs are available.

The engine retrieves the compiled predicate from the cache (or compiles it if not cached). It then prepares the input arguments by collecting the step's input attributes from the flow's current state.

**For Lua predicates**:
1. Acquire a Lua state from the pool
2. Load the predicate bytecode
3. Push input arguments onto the stack in sorted order
4. Execute the predicate with protected call (to catch errors)
5. Extract the return value
6. Convert to boolean (Lua's truthiness rules apply)
7. Return the state to the pool

**For Ale predicates**:
1. Build an argument vector from sorted input names
2. Call the procedure with those arguments
3. The result is truthy unless it's the literal value `false`

If the predicate returns true, the step proceeds to execution. If it returns false, the step is marked as "skipped" and its downstream dependencies are re-evaluated to see if they can still proceed without this step's outputs.

If predicate evaluation fails (throws an error), the step is marked as failed.

### How Scripts Are Executed

Script execution is similar to predicate evaluation but produces outputs instead of a boolean.

The engine retrieves the compiled script and prepares input arguments the same way as predicates.

**For Lua scripts**:
1. Acquire a state from the pool
2. Load the bytecode
3. Push arguments
4. Execute with protected call
5. Extract the return value from the stack
6. Convert to Go types (maps and primitives)
7. Return the state to the pool

If the script returns a Lua table, it's converted to a Go map and treated as named outputs. If it returns a single value, it's wrapped in a map with the key "result".

**For Ale scripts**:
1. Build argument vector
2. Call the procedure
3. Extract and convert the result to Go types

The result is automatically converted from Ale data structures to Go types.

The outputs are then added to the flow's attribute state, making them available to downstream steps.

### Type Conversion

The engine automatically converts between Go types (used by the engine) and script language types.

**Lua type conversions**:
- Go strings ↔ Lua strings
- Go booleans ↔ Lua booleans
- Go integers ↔ Lua integers
- Go floats ↔ Lua numbers
- Go slices ↔ Lua array tables (sequential integer keys)
- Go maps ↔ Lua tables (string keys)
- Go nil ↔ Lua nil

**Ale type conversions**:
- Go strings ↔ Ale String
- Go booleans ↔ Ale Bool
- Go integers ↔ Ale Integer
- Go floats ↔ Ale Float
- Go slices ↔ Ale Vector
- Go maps ↔ Ale Object
- Go nil ↔ Ale Null

These conversions happen automatically when passing inputs to scripts and extracting outputs, so users don't need to think about marshaling.

### Error Handling

Compilation errors are caught during step registration/update and prevent the step from being stored.

Runtime errors during predicate evaluation or script execution are caught and result in step failure. The error message is stored in the step's execution state and can be inspected for debugging.

For Lua, the protected call mechanism catches panics and runtime errors. For Ale, a panic recovery wrapper catches any errors during procedure calls.

All errors are propagated up to the flow execution layer, which decides how to handle them (retry, fail the step, fail the flow, etc.).

---

## Plan Generation and Preview

### What is an Execution Plan?

An execution plan is a blueprint that answers the question: "What steps need to run, and in what order, to achieve my goals?"

When you want to execute a flow, you specify one or more goal steps (the outcomes you want). The engine then works backwards from those goals to figure out:

- Which other steps must run to produce the inputs that the goal steps need
- Which other steps must run to produce the inputs that *those* steps need
- And so on, recursively, until it reaches steps whose inputs are already available

The result is a directed acyclic graph (DAG) of dependencies: a plan that includes all the steps needed to reach your goals, with no circular dependencies and no unnecessary steps.

### The Components of a Plan

A plan contains:

**Goals**: The step IDs you explicitly requested. These are the "leaf nodes" you want to reach.

**Steps**: All the steps that need to execute, including the goals and all their transitive dependencies.

**Attributes**: A dependency graph showing which steps produce each attribute (providers) and which steps consume each attribute (consumers).

**Required Inputs**: The attributes that must be provided externally (in the flow's initial state) because no step in the plan produces them.

### How Plan Generation Works

Plan generation uses a two-pass approach over the dependency graph.

#### Initialization

The engine starts by creating a plan builder with:

- A **visited set** to track steps that have already been processed (prevents infinite loops)
- An **available set** of attribute names that are present in the initial state
- A **missing set** to collect required inputs that couldn't be satisfied
- An **attributes map** to build the provider-consumer relationships for this specific plan
- A reference to **EngineState.Attributes** containing the pre-computed dependency graph

#### Pass 1: Satisfiable Steps

The engine first computes which steps are satisfiable given the initial state and any outputs that can be produced by other satisfiable steps. This pass does not build the plan; it only determines which steps could run if needed.

#### Pass 2: Goal Traversal

For each goal step, the engine walks upstream to find providers for required and optional inputs:

1. **Checks if already visited**: If this step was already processed, return immediately to avoid duplicate work.

2. **Processes inputs**: For each required or optional input:
   - If the attribute is already in the initial state, no provider is needed
   - Otherwise, the engine looks up providers for that attribute
   - Only providers deemed satisfiable in pass 1 are included
   - If a required input has no satisfiable provider, it is added to the missing set

3. **Includes or excludes the step**: A step is included if any of its output attributes are not already present in the initial state. Steps whose outputs are fully satisfied by the initial state are excluded.

#### Provider Discovery

When a step needs an input attribute, the engine looks up all providers for that attribute using the cached dependency graph from EngineState.Attributes. This map is maintained as a projection—automatically rebuilt whenever steps are registered, updated, or unregistered—ensuring O(1) lookup during plan generation.

If multiple steps can provide the same attribute, **all satisfiable providers are included in the plan**. During execution, these providers compete—whichever completes first provides the attribute value. This enables redundancy and allows the fastest provider to win.

The dependency relationship is bidirectional: the consuming step is added to the attribute's consumers list, and all providing steps are added to the attribute's providers list.

#### Handling Optional Inputs

Optional inputs are treated differently from required inputs. If an optional input isn't available and no provider exists, that's fine—the step will use its default value if one is defined, otherwise the input is omitted. The attribute isn't added to the missing set.

However, if a provider *does* exist for an optional input, the provider is included in the plan just like for required inputs. This ensures that if the data can be computed, it will be.

### Plan Completeness

The plan generation algorithm includes all steps needed to reach the goals, including all satisfiable providers for each required attribute.

If a step's outputs are already available in the initial state, the step is excluded from the plan. For example, if you provide `{"user_id": 123}` in the initial state, any step that produces "user_id" won't be included in the plan.

Steps are only included if they are reachable from the goals via provider traversal. Steps outside that traversal are not part of the plan.

When multiple steps can provide the same attribute, all satisfiable providers are included to enable competition during execution. This is not strictly minimal, but ensures redundancy and allows the fastest provider to satisfy the dependency.

### Cycle Prevention

Circular dependencies are prevented at step registration time, so by the time plan generation runs, cycles are impossible. The registration-time validation uses `api.BuildDependencies` to construct a temporary attribute dependency map that includes all registered steps plus the new step being validated, then performs depth-first search with a recursion stack starting from the new step to detect cycles.

If a cycle exists—for example, step A depends on B, B depends on C, and C depends on A—the registration of the step that closes the cycle will fail.

This guarantee means plan generation doesn't need to worry about cycles; it can safely assume the dependency graph is acyclic and can rely on the cached EngineState.Attributes for efficient provider lookups.

### How Plans Are Used During Execution

Once a plan is generated, it becomes the blueprint for flow execution.

The flow state contains the plan, and throughout execution, the engine uses the plan to make decisions:

**Finding initial steps**: Steps whose inputs are all available in the flow's initial state. These can start immediately.

**Finding ready steps**: After a step completes, the engine looks at which attributes it produced, then uses the plan's attribute map to find downstream consumers of those attributes. Those consumers become candidates for execution if their other inputs are also satisfied.

**Determining necessity**: Before executing a step, the engine checks if it's a goal step or if its outputs are needed by any pending downstream steps. If neither is true, the step is skipped with the reason "outputs not needed".

**Completion detection**: The flow is complete when all steps in the plan have either completed successfully or been skipped.

### What is Plan Preview?

Plan preview is a read-only operation that generates a plan without executing it. It's used by the UI to show users what would happen if they executed a particular step.

When you click a step in the UI, the frontend calls the plan API with that step as the goal and (optionally) some initial state. The engine generates the plan and returns it as JSON.

Plan preview responses include an `excluded` map for steps omitted due to missing required inputs or outputs already present in the initial state. The map groups excluded steps under `missing` (with required inputs) and `satisfied` (with outputs already in the initial state).

The UI then uses this plan to visually highlight:
- The goal step (what you clicked)
- All steps that would execute to reach the goal
- All steps that wouldn't execute (grayed out)
- The dependency edges between steps in the plan

This gives users a preview of execution before they commit to running the flow.

#### Preview vs Execution

Preview is purely computational—no state changes, no side effects. It's a "what if" operation.

Execution uses the plan to actually run steps, produce outputs, and update flow state.

The same plan generation algorithm is used for both preview and execution, ensuring that what you see in the preview is exactly what will execute.

However, some dynamic aspects aren't captured in preview:
- Predicates aren't evaluated (the plan assumes all steps will execute)
- Work items aren't computed (parallel execution isn't shown)
- Runtime failures aren't predicted

So the preview shows the "optimistic path"—what will happen if all steps succeed and all predicates return true.

### Example: Planning from Goals

Imagine these registered steps:
- **Step A**: Outputs "customer_id"
- **Step B**: Requires "customer_id", outputs "order_list"
- **Step C**: Requires "order_list", outputs "total_value"
- **Step D**: Requires "total_value", outputs "recommendation"

You request a flow with goal step D and initial state `{}`.

The plan generation process:

1. Process goal D:
   - D requires "total_value"
   - Find provider: Step C produces "total_value"
   - Recursively process C

2. Process C (from D's dependency):
   - C requires "order_list"
   - Find provider: Step B produces "order_list"
   - Recursively process B

3. Process B (from C's dependency):
   - B requires "customer_id"
   - Find provider: Step A produces "customer_id"
   - Recursively process A

4. Process A (from B's dependency):
   - A requires nothing
   - Add A to plan
   - Mark "customer_id" as available

5. Back to B: Add to plan, mark "order_list" as available

6. Back to C: Add to plan, mark "total_value" as available

7. Back to D: Add to plan, mark "recommendation" as available

Final plan:
- **Goals**: [D]
- **Steps**: {A, B, C, D}
- **Attributes**:
  - "customer_id": providers=[A], consumers=[B]
  - "order_list": providers=[B], consumers=[C]
  - "total_value": providers=[C], consumers=[D]
  - "recommendation": providers=[D], consumers=[]
- **Required**: []

If instead the initial state was `{"customer_id": 123}`:

1. Process goal D (same as before down to...)

2. Process B:
   - B requires "customer_id"
   - "customer_id" is in available set (from initial state)
   - Don't need to find provider, mark B as consumer of "customer_id"

3. Process A:
   - A's output "customer_id" is already available
   - Mark A as visited but don't add to plan

Final plan:
- **Goals**: [D]
- **Steps**: {B, C, D}  (A excluded)
- **Required**: []  (already provided in initial state)

If no step provides "customer_id" and the initial state is `{}`:

Final plan:
- **Goals**: [D]
- **Steps**: {B, C, D}
- **Required**: ["customer_id"]  (must be provided in initial state)

---

## Flow Execution

### What is a Flow?

A flow is a live instance of an execution plan. When you start a flow, you're saying "execute this plan with this initial state". The flow tracks the runtime state of each step as they execute, accumulate outputs, and progress toward completion.

Think of the execution plan as a recipe, and the flow as the actual cooking process. The plan says what steps to do and in what order; the flow is the real-time tracking of which steps are done, which are in progress, which succeeded, which failed, and what data has been produced.

### Flow State

A flow maintains several pieces of state:

**The plan**: The immutable blueprint generated from goals and initial state. This doesn't change during execution.

**Step executions**: For each step in the plan, the flow tracks its status (pending, active, completed, failed, skipped), when it started, when it finished, what inputs it received, what outputs it produced, and any error messages.

**Attributes**: A global key-value map of all attributes produced during the flow. When a step completes, its outputs are added here, making them available to downstream steps.

**Flow status**: Whether the flow itself is active, completed, or failed.

### Flow Event Processing

Flows advance when events are raised against their aggregates. Events come from API entrypoints (StartFlow, work completions), step execution (sync work completion), async callbacks (webhook completions), child flow completions, and retry timers. Each event is applied to the flow aggregate using the timebox executor with optimistic concurrency.

Flows can have many steps executing concurrently, producing outputs and triggering downstream steps. Without careful synchronization, this could lead to race conditions and inconsistent state. The engine avoids this by applying each event inside a transaction and letting the executor retry on version conflicts. State transitions are derived from the event log, not in-place mutation.

Work items are keyed by tokens. When a completion arrives, the token identifies the specific work item to update inside the flow's execution state.

### Engine Startup and Recovery

When the engine starts, it performs recovery before accepting new flows.

**Recovery** means finding all flows that were active when the engine last shut down and resuming them. The engine does this by reading the event log and identifying flows with a "FlowStarted" event but no terminal event (completed or failed).

For each active flow, the engine:

1. Reconstructs the flow state by replaying all events from the log
2. Identifies work items that were in "active" state (meaning they were executing when the engine shut down)
3. Re-executes those work items to complete them
4. Identifies work items that have scheduled retries in the past (retries that should have happened while the engine was down)
5. Immediately retries those work items

This ensures that flows continue from where they left off, without losing any work or requiring manual intervention.

After recovery completes, the engine starts a background retry loop that uses a timer-based queue. When a `RetryScheduledEvent` is recorded for a work item, the retry is added to the queue. The loop sleeps until the earliest retry time, then executes all ready retries.

### Starting a Flow

When you call the StartFlow API, several things happen:

**Plan generation**: The engine generates an execution plan from your goal steps and initial state using the algorithm described earlier.

**Input validation**: The engine checks that all required inputs (those in the plan's "Required" list) are present in the initial state. If any are missing, the flow is rejected.

**Event emission**: A "FlowStartedEvent" is created and persisted to the event log. This event contains the plan and initial state.

**Initial step discovery**: The FlowStartedEvent is applied in a transaction, which finds all steps that can start immediately (those whose inputs are all available in the initial state) and prepares them for execution. Work execution is scheduled after the transaction commits.

At this point, the flow is active and steps begin executing.

### Step Readiness and Scheduling

A step can only start executing when three conditions are met:

**Its status is "pending"**. Steps start in pending status. Once they've started, they transition to active, then eventually to completed, failed, or skipped (terminal states).

**All required inputs are available**. The engine checks if every attribute marked as "required" in the step's specification is present in the flow's attribute map. If any are missing, the step can't start.

**Its outputs are needed**. This is an optimization: if a step's outputs won't be used by any downstream steps, there's no point in executing it. The engine checks if the step is a goal step (always execute) or if any of its outputs are consumed by steps that are still pending.

The engine finds ready steps in two scenarios:

**Initial steps**: When the flow starts, the engine scans all steps in the plan and finds those whose inputs are satisfied by the initial state.

**Downstream steps**: When a step completes, the engine looks at which attributes it produced, then uses the plan's dependency graph to find steps that consume those attributes. Those steps become candidates for execution.

### Step Lifecycle

When a step is ready to execute, the engine walks it through the lifecycle below.

#### Input Collection

The engine gathers all the step's input attributes from the flow's attribute map. This includes both required and optional inputs. If an optional input is missing, the step's default value is used if one is specified. Otherwise the input is omitted.

#### Output Necessity Check

Before executing, the engine checks whether the step is a goal step or whether any pending downstream steps still need one of its outputs. If neither is true, the step is skipped with the reason "outputs not needed".

#### Predicate Evaluation

If the step has a predicate, it's evaluated with the collected inputs. If the predicate returns false, the step is skipped with reason "predicate returned false". If predicate evaluation throws an error, the step fails.

#### Work Item Computation

This is where parallel execution is handled. Some steps can execute multiple times in parallel with different input values, based on the "for_each" attribute flag.

The engine identifies which attributes are marked as "for_each". If the input value for a for_each attribute is an array, the engine uses it to compute the Cartesian product that creates individual work items. If a for_each input is not an array, it behaves like a normal scalar input and does not create multiple work items on its own.

For example, if a step has:
- `users` (for_each): ["alice", "bob", "charlie"]
- `action`: "notify"

Three work items are created:
- Item 1: {users: "alice", action: "notify"}
- Item 2: {users: "bob", action: "notify"}
- Item 3: {users: "charlie", action: "notify"}

If there are multiple for_each attributes, all combinations are generated. With `users` ["alice", "bob"] and `actions` (for_each) ["notify", "log"], you'd get four work items.

If there are no for_each attributes, a single work item is created with all the step's inputs.

Each work item gets a unique token (UUID) to identify it.

#### Event Emission and Deferred Execution

All the above preparation happens inside a database transaction to ensure atomicity. Once preparation completes, a StepStartedEvent is emitted within the transaction.

The transaction also returns a deferred function—a callback that will execute the actual work after the transaction commits.

Why deferred? Because step execution can be long-running (HTTP calls, scripts that take time), and we don't want to hold database locks during that time. By deferring the work until after the transaction commits, we keep transactions short and avoid blocking other flows.

After the transaction commits and the event is persisted, the deferred function is called, which actually executes the work items.

### Work Item Execution

Work items execute concurrently, controlled by the step's parallelism setting.

The engine creates a semaphore channel with capacity equal to the parallelism value (default 1, meaning serial execution). Before each work item executes, it must acquire the semaphore. When done, it releases the semaphore.

This means if parallelism is 5, up to 5 work items can execute concurrently. If parallelism is 1, work items execute one at a time.

For each work item:

1. **Re-evaluate the predicate**: Even though the step-level predicate was checked earlier, it's evaluated again for each work item with that item's specific inputs. This allows fine-grained control (e.g., skipping only certain items).

2. **Emit WorkStartedEvent**: Marks the work item as active.

3. **Perform the actual work**:
   - For script steps: Execute the compiled script with the work item's inputs
   - For HTTP steps: Make an HTTP request to the endpoint with the inputs as payload

4. **Handle the result**:
   - If successful, emit WorkSucceededEvent with the outputs
   - If failed, emit WorkFailedEvent with the error
   - If retriable (temporary failure), emit WorkNotCompletedEvent to trigger retry logic

#### HTTP Execution

For synchronous HTTP steps, the engine makes an HTTP POST request to the configured endpoint with the inputs as JSON in the body. The response is expected to contain the outputs as JSON.

For asynchronous HTTP steps, the engine includes a webhook URL in the request metadata. The endpoint is expected to call this webhook later when the work completes. The engine doesn't wait for the response; instead, it marks the work item as active and waits for the webhook callback.

When the webhook is called, it triggers a "CompleteWork" operation that records the outputs and marks the work item as succeeded.

### Output Aggregation and Attribute Setting

When all work items for a step complete successfully, the engine aggregates their outputs.

**If there was a single work item**: Its outputs become the step's outputs directly.

**If there were multiple work items**: Outputs are grouped by attribute name. For each output attribute, the engine creates an array containing all the values produced by the work items, along with metadata about which for_each values produced each output.

#### Important: Output Order Is Not Preserved

**Work items execute in parallel, and the output array order does NOT match the input array order.** You cannot rely on `output[0]` corresponding to `input[0]`.

The order is arbitrary because:
1. Work items complete at different times (the first input might finish last)
2. Results are collected from a map structure with undefined iteration order

Instead of relying on position, use the metadata included with each output to match results back to inputs.

#### Output Structure Example

Consider a step that sends notifications to users:
- Input: `users` (for_each): `["alice", "bob", "charlie"]`
- Output attribute: `message_id`

When all three work items complete, the aggregated output looks like:

```json
{
  "message_id": [
    {
      "users": "bob",
      "message_id": "msg-456"
    },
    {
      "users": "charlie",
      "message_id": "msg-789"
    },
    {
      "users": "alice",
      "message_id": "msg-123"
    }
  ]
}
```

Notice that:
- The order is `["bob", "charlie", "alice"]`, not `["alice", "bob", "charlie"]`
- Each entry includes the `users` value that produced it (the metadata)
- **The output value uses the output attribute name as its key** (`message_id` in this case)
- You can match outputs to inputs by checking the metadata fields, not by array position

#### How Output Naming Works

Each entry in the aggregated array contains:
1. **Metadata fields**: One key for each for_each input attribute (e.g., `"users": "bob"`)
2. **Output field**: One key using the output attribute name (e.g., `"message_id": "msg-456"`)

This naming scheme prevents collisions: if an input attribute were named "message_id", it would have a different role (input vs output) and wouldn't appear in the for_each metadata. The engine uses the actual output attribute name as the key, ensuring:
- No generic "value" key that could conflict with input attributes
- Semantic clarity (the key name tells you what the data represents)
- Consistent naming at both the outer level (`"message_id": [...]`) and inner level (`"message_id": "msg-456"`)

If your step has multiple for_each attributes, the metadata includes all of them:

```json
{
  "result": [
    {
      "users": "bob",
      "actions": "notify",
      "result": "notification sent"
    },
    {
      "users": "alice",
      "actions": "log",
      "result": "logged"
    }
  ]
}
```

#### Attribute Events

The aggregated outputs are then added to the flow's attribute map via "AttributeSetEvent" events. Each attribute gets its own event.

Once attributes are set, downstream steps that consume them may become ready to execute.

### Step Completion, Failure, and Downstream Discovery

When all work items for a step have reached terminal states (succeeded or failed), the step itself transitions to a terminal state:

**Completed**: If all work items succeeded.

**Failed**: If any work item failed permanently.

A StepCompletedEvent or StepFailedEvent is emitted.

Additional failure cases:
- **Predicate evaluation error**: The predicate threw an error during evaluation.
- **Required inputs become unreachable**: A pending step loses all viable providers for a required input, producing the error "required input no longer available".

Steps excluded during plan generation are not skipped; they simply never appear in the plan.

The engine then looks for downstream steps by:

1. Identifying which attributes this step produced
2. Using the plan's dependency graph to find steps that consume those attributes
3. Checking if those steps are now ready to execute
4. If ready, preparing and starting them

This cascading discovery continues until all reachable steps have executed.

### Error Handling and Retry Logic

When a work item fails, the engine must decide whether to retry or fail permanently.

**Retry decision**: The engine checks the step's work configuration for a maximum retry count. If the work item hasn't exceeded this count, it's eligible for retry.

**Backoff calculation**: The engine calculates when to retry based on the backoff strategy:
- **Fixed**: Same delay every time (e.g., always retry after 10 seconds)
- **Linear**: Delay increases linearly with attempt count (e.g., 10s, 20s, 30s...)
- **Exponential**: Delay doubles each time (e.g., 10s, 20s, 40s, 80s...)

**Retry scheduling**: A "RetryScheduledEvent" is emitted with the next retry time. The work item's status becomes "pending" again, and its retry count is incremented.

**Retry execution**: When a `RetryScheduledEvent` is received, it's added to the retry queue. The retry loop sleeps until the earliest retry time arrives, then executes all ready work. This timer-based approach is more efficient than polling.

**Permanent failure**: If a work item exceeds the maximum retry count, it's marked as permanently failed. This causes the step to fail, which may cascade to dependent steps.

### Failure Propagation

When a step fails, the engine determines which other steps are now unreachable (cannot complete because they depend on the failed step's outputs).

The algorithm:

1. For each step still in "pending" status, check if all its required inputs can be satisfied
2. An input can be satisfied if it's already in the attribute map, or if there's a provider step that hasn't failed
3. If any required input cannot be satisfied, mark the step as failed with an error indicating dependency failure

This cascades recursively: failing steps may cause other steps to become unreachable, which fail them, which may cause more failures, and so on.

**Goal failure**: If any goal step fails or becomes unreachable, the entire flow is marked as failed.

### Flow Completion

A flow reaches completion when one of two conditions is met:

**Success**: All goal steps have completed successfully. The flow transitions to "completed" status and a "FlowCompletedEvent" is emitted.

**Failure**: Any goal step cannot complete (either failed directly or became unreachable due to dependency failures). The flow transitions to "failed" status and a "FlowFailedEvent" is emitted.

In both cases, the flow is terminal—no more steps will execute, and the flow will be deactivated once no active work remains.

### Event Sourcing and State Reconstruction

The engine uses event sourcing, which means the flow state is never directly modified. Instead, all changes are recorded as events, and the state is the result of applying all events in order.

#### Flow Events

Every operation that changes flow state emits one or more events. Here are examples of each event type:

**FlowStartedEvent** - Emitted when a flow begins execution:
```json
{
  "type": "flow_started",
  "timestamp": "2025-12-01T14:00:00Z",
  "data": {
    "flow_id": "flow-123",
    "plan": {
      "goals": ["send-email"],
      "steps": {
        "fetch-user": {...},
        "send-email": {...}
      },
      "attributes": {
        "user_id": {"providers": [], "consumers": ["fetch-user"]},
        "email": {"providers": ["fetch-user"], "consumers": ["send-email"]},
        "message_id": {"providers": ["send-email"], "consumers": []}
      },
      "required": ["user_id"]
    },
    "init": {
      "user_id": "12345"
    }
  }
}
```

**StepStartedEvent** - Emitted when a step begins (after predicate evaluation):
```json
{
  "type": "step_started",
  "timestamp": "2025-12-01T14:00:01Z",
  "data": {
    "flow_id": "flow-123",
    "step_id": "fetch-user",
    "inputs": {
      "user_id": "12345"
    },
    "work_items": {
      "token-abc": {"user_id": "12345"}
    }
  }
}
```

**WorkStartedEvent** - Emitted when a work item begins execution:
```json
{
  "type": "work_started",
  "timestamp": "2025-12-01T14:00:01.100Z",
  "data": {
    "flow_id": "flow-123",
    "step_id": "fetch-user",
    "token": "token-abc"
  }
}
```

**WorkSucceededEvent** - Emitted when a work item completes successfully:
```json
{
  "type": "work_succeeded",
  "timestamp": "2025-12-01T14:00:02Z",
  "data": {
    "flow_id": "flow-123",
    "step_id": "fetch-user",
    "token": "token-abc",
    "outputs": {
      "email": "user@example.com",
      "name": "Alice"
    }
  }
}
```

**WorkFailedEvent** - Emitted when a work item fails permanently:
```json
{
  "type": "work_failed",
  "timestamp": "2025-12-01T14:00:02Z",
  "data": {
    "flow_id": "flow-123",
    "step_id": "fetch-user",
    "token": "token-abc",
    "error": "database connection timeout"
  }
}
```

**WorkNotCompletedEvent** - Emitted when a work item fails but will be retried:
```json
{
  "type": "work_not_completed",
  "timestamp": "2025-12-01T14:00:02Z",
  "data": {
    "flow_id": "flow-123",
    "step_id": "fetch-user",
    "token": "token-abc",
    "error": "temporary network error"
  }
}
```

**RetryScheduledEvent** - Emitted when a work item is scheduled for retry:
```json
{
  "type": "retry_scheduled",
  "timestamp": "2025-12-01T14:00:02.100Z",
  "data": {
    "flow_id": "flow-123",
    "step_id": "fetch-user",
    "token": "token-abc",
    "retry_count": 1,
    "next_retry_at": "2025-12-01T14:00:12Z"
  }
}
```

**AttributeSetEvent** - Emitted when step outputs are added to flow state:
```json
{
  "type": "attribute_set",
  "timestamp": "2025-12-01T14:00:02.200Z",
  "data": {
    "flow_id": "flow-123",
    "name": "email",
    "value": "user@example.com",
    "provider": "fetch-user"
  }
}
```

**StepCompletedEvent** - Emitted when all work items for a step succeed:
```json
{
  "type": "step_completed",
  "timestamp": "2025-12-01T14:00:02.300Z",
  "data": {
    "flow_id": "flow-123",
    "step_id": "fetch-user",
    "outputs": {
      "email": "user@example.com",
      "name": "Alice"
    },
    "duration": 1200
  }
}
```

**StepFailedEvent** - Emitted when a step fails:
```json
{
  "type": "step_failed",
  "timestamp": "2025-12-01T14:00:02Z",
  "data": {
    "flow_id": "flow-123",
    "step_id": "fetch-user",
    "error": "database connection timeout"
  }
}
```

**StepSkippedEvent** - Emitted when a step is skipped (predicate returned false or outputs not needed):
```json
{
  "type": "step_skipped",
  "timestamp": "2025-12-01T14:00:01.500Z",
  "data": {
    "flow_id": "flow-123",
    "step_id": "send-sms",
    "reason": "predicate returned false"
  }
}
```

**FlowCompletedEvent** - Emitted when all plan steps complete successfully or are skipped:
```json
{
  "type": "flow_completed",
  "timestamp": "2025-12-01T14:00:05Z",
  "data": {
    "flow_id": "flow-123",
    "duration": 5000
  }
}
```

**FlowFailedEvent** - Emitted when any goal step fails or becomes unreachable:
```json
{
  "type": "flow_failed",
  "timestamp": "2025-12-01T14:00:03Z",
  "data": {
    "flow_id": "flow-123",
    "error": "goal step 'send-email' cannot complete: required input 'email' unavailable"
  }
}
```

#### Event Application

Each event has an "applier" function that takes the current state and the event data and returns a new state.

When the engine needs the current flow state, it replays all events through their appliers to reconstruct the state. This happens efficiently using the timebox library, which maintains materialized views of the state.

The benefits:

**Auditability**: The complete history of the flow is available by reading the event log.

**Recovery**: After a crash, the engine replays events to reconstruct exactly where each flow was.

**Debugging**: You can inspect the event sequence to understand what happened during execution.

**Time travel**: You can reconstruct the state at any point in time by applying events up to that timestamp.

### Transactional Guarantees

All state changes happen within database transactions, ensuring atomicity and consistency.

The pattern is:
1. Read current state
2. Apply business logic to compute events
3. Persist events to the log
4. Apply events to update state
5. Commit transaction

The timebox library provides optimistic locking: if two concurrent operations try to modify the same flow, one will succeed and the other will retry with the updated state.

Deferred execution (the actual step work) happens outside the transaction to avoid holding locks during long-running operations.

### Concurrency Model Summary

The engine achieves high concurrency while maintaining safety through several mechanisms:

**Event-sourced state**: All state changes are events applied to aggregates.

**Optimistic concurrency**: Aggregate updates are transactional with automatic retry on conflicts.

**Work item parallelism**: Within a step, multiple work items execute concurrently, controlled by semaphore.

**Transaction isolation**: State changes are atomic and isolated via database transactions.

**Deferred execution**: Long-running work happens outside transactions to avoid blocking.

This architecture allows:
- Multiple flows executing concurrently (true parallelism across flows)
- Serial processing within each flow (safety and consistency)
- Parallel work items within each step (performance optimization)
- No manual locking required (serial flow processing handles synchronization)

### The Complete Execution Flow

To tie it all together, here's the journey of a flow from start to finish:

1. **StartFlow called** with goals and initial state
2. **Plan generated** by recursive dependency resolution
3. **FlowStartedEvent emitted** and persisted
4. **FlowStartedEvent applied** to the flow aggregate
5. **Initial steps discovered** (inputs satisfied by initial state)
6. **Each initial step prepared**: inputs collected, predicate evaluated, work items computed
7. **Work items execute** in parallel (controlled by parallelism limit)
8. **Work completes**, emitting WorkSucceededEvents
9. **Outputs aggregated** and added to flow attributes
10. **Downstream steps discovered** using dependency graph
11. **Steps 6-10 repeat** for downstream steps
12. **Eventually all reachable steps complete**
13. **Flow completion checked**: all goals completed?
14. **FlowCompletedEvent emitted**
15. **Flow deactivated** once no active work remains

If any step fails and is a goal or causes a goal to become unreachable, the flow fails instead of completing.

If any work item needs retry, it's scheduled and re-executed later, potentially delaying completion.

This entire process is driven by events applied to aggregates, coordinated by transactional updates and deferred work execution, and persisted through event sourcing.
