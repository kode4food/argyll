# Core Concepts

## Args vs Attributes

**Arguments (Args)**
- Input/output parameters for step execution
- Type: `Args` = `map[Name]any`
- Scope: Individual step execution

**Attributes**
- Flow state accumulated from step outputs with provenance tracking
- Type: `map[Name]*AttributeValue` (value + producing step ID)
- Scope: Entire flow lifecycle

**Data Flow:**
```
Step Outputs → Flow Attributes (with provenance)
                        ↓
Flow Attributes → Next Step Inputs
```

**Naming:**
- Backend: `Attributes map[Name]*AttributeSpec` (step definitions)
- Backend: `Attributes map[Name]*AttributeValue` (flow state)
- Backend: `Args` (runtime values)
- Frontend: `satisfiedArgs`/`timedOutArgs` (step execution tracking)
- Frontend: `attributeProvenance` (flow state tracking)

## Goal-Oriented Execution

Flows specify **Goal Steps** - the targets to reach. The engine:

1. Walks backward from all goal steps
2. Creates execution plan as union of required steps
3. Determines required inputs for flow start
4. Executes steps in dependency order

**Multiple Goals:**
- Plan includes all steps needed for any declared goal
- Goals may complete in any order
- Lazy evaluation still applies

## Step Types

- **Sync Steps**: HTTP endpoints returning results immediately
- **Async Steps**: HTTP endpoints with webhook callback for completion
- **Script Steps**: Ale or Lua scripts executed in-engine
- **Flow Steps**: Sub-flows executed by the engine using child flow goals and optional input/output mapping

**Step Declaration:**
- Required inputs: Must be available before execution
- Optional inputs: Use defaults if not provided by upstream
- Outputs: Values produced by the step
- Predicate: Optional condition for execution (Ale/Lua)

## Step Patterns (Conceptual)

Design guidance, not enforced types:

- **Resolver**: No required inputs, produces outputs on demand
- **Processor**: Takes inputs, produces outputs (transformation)
- **Collector**: Takes inputs, no outputs (side effects)

## Lazy Evaluation

Only steps in the execution plan execute. Benefits:

- Minimal resource consumption
- Reduced execution time
- Lower complexity
- Fewer failure points
