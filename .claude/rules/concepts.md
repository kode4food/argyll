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

## Memoizable

Memoizable is an optional property on steps that enables result caching at the engine level:

**How It Works:**
- When a step is marked as memoizable, the engine maintains a global cache of (step definition, inputs) → outputs mappings
- Before executing the step, the engine checks the cache using a deterministic key derived from the step's functional definition and input arguments
- Cache hit: Returns cached outputs without executing the step
- Cache miss: Executes the step normally and stores the result in the cache
- Only successful executions are cached; failures are never cached

**Key Characteristics:**
- **Scope**: Individual work items (not aggregated step results)
- **Key Construction**: SHA256 hash of (step definition snapshot + sorted input args)
- **Step Definition Hash**: Includes only functional fields (Type, Attributes, Config); excludes Name, ID, Labels
- **Cache Eviction**: LRU (Least Recently Used) with configurable size
- **Deterministic**: Identical step configurations always produce identical hashes, even across engine restarts
- **Thread-Safe**: Protected by RWMutex; safe for concurrent access

**When to Use Memoizable:**
- Steps with expensive computations (heavy calculations, external API calls with latency)
- Steps with deterministic outputs (same inputs always produce same outputs)
- Steps that are executed repeatedly with overlapping inputs

**When NOT to Use:**
- Steps with side effects (database writes, external state changes)
- Steps with non-deterministic outputs (timestamps, random values)
- Steps that depend on external state beyond their inputs

**Configuration:**
- Set via builder: `step.WithMemoizable()`
- Set via API: Include `"memoizable": true` in step definition
- Set via Web UI: Toggle "Cache step results (memoizable)" in Execution Options
- Cache size: Configured via `MEMO_CACHE_SIZE` environment variable (default: 4096 entries)

**Limitations:**
- Cache is in-memory and lost on engine restart
- No TTL (time-to-live); entries remain until evicted by LRU policy
- For_each loops cache individual work items, not aggregated results
