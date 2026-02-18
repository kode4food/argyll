# Memoization (Result Caching)

**Memoization** enables optional result caching at the engine level. When enabled, the engine caches step results and returns cached outputs for identical inputs, avoiding redundant execution.

## How It Works

1. **Mark a step as memoizable:**
   ```json
   {
     "id": "calculate-discount",
     "memoizable": true,
     "type": "sync",
     "http": { "endpoint": "..." }
   }
   ```

2. **First execution with inputs `{amount: 100, rate: 0.1}`:**
   - Engine checks cache → miss
   - Step executes normally
   - Outputs cached: `{discount_amount: 90}`

3. **Second execution with same inputs `{amount: 100, rate: 0.1}`:**
   - Engine checks cache → hit
   - Returns cached outputs immediately
   - Step handler never called

4. **Execution with different inputs `{amount: 200, rate: 0.1}`:**
   - Different inputs → cache miss
   - Step executes
   - New result cached

## Cache Key

The cache key is deterministic, derived from:

1. **Step definition** (only functional fields):
   - Type (sync, async, script)
   - HTTP config, Script code, Flow config
   - Attributes specification
   - WorkConfig (retry, parallelism settings)

2. **Input arguments** (sorted for consistency)

**Important:** The cache key includes the step's code/configuration. If you update the step definition, the cache is automatically invalidated (different key).

**Example:**

```
Step: calculate_discount
Config hash: abc123def (from step definition)
Inputs: {amount: 100, discount_rate: 0.1}
Input hash: xyz789 (sorted, deterministic)

Cache key: abc123def:xyz789
```

Two different step definitions with the same inputs will have different cache keys and won't share cache entries.

## Scope

Cache is **per-work-item**, not aggregated across all work items of a step with `for_each`.

```
Step: process-item [for_each, memoizable]
Inputs:
  items: [item-1, item-2, item-3]

Cache keyed as:
  step_hash:item-1-hash → result-1 (cache miss, execute)
  step_hash:item-2-hash → result-2 (cache miss, execute)
  step_hash:item-1-hash → result-1 (cache hit, return cached)

Each work item is cached independently.
```

## Cache Lifecycle

### In-Memory

- Cache is in-memory (in the engine process)
- Lost on engine restart
- **Not persisted** to Redis

### LRU Eviction

- Cache uses LRU (Least Recently Used) eviction
- Configurable size (default: 10240 entries)
- When full, least recently used entries are removed

### No TTL

- Entries remain until evicted by LRU
- No time-based expiration
- Suitable for data that doesn't change frequently

### Cache Scope

- Each engine instance has its own cache (not shared across instances)
- Cache is lost on engine restart

## When to Use Memoization

### Do Use ✓

- **Expensive computations**: Calculations that take significant CPU
- **Slow external API calls**: Calling slow third-party services
- **Deterministic outputs**: Same inputs always produce same outputs
- **Repeated patterns**: Flow often executes the same step with same inputs

**Example:**
```
Step: lookup-exchange-rate
- Call external API (slow, reliable)
- Same currency pair → same rate (deterministic)
- Memoizable: YES
```

### Don't Use ✗

- **Non-deterministic outputs**: Timestamps, random values, current state
- **Side effects**: Database writes, file uploads, external state changes
- **Time-sensitive data**: Stock prices, weather, real-time data
- **One-time operations**: Rarely executed with same inputs

**Example:**
```
Step: process-payment
- Makes payment (has side effects)
- Current flow state affects result
- Memoizable: NO (would skip actual payment on retry!)
```

## Configuration

### Enable Memoization

**Via API:**
```json
{
  "id": "lookup-exchange-rate",
  "memoizable": true,
  "type": "sync",
  "http": { "endpoint": "https://api.example.com/rates" }
}
```

**Via Go SDK:**
```go
step := builder.
  NewStep().WithName("lookup-exchange-rate").
  WithMemoizable().
  WithSync("https://api.example.com/rates")
```

**Via Web UI:**
Check "Cache step results (memoizable)" in step execution options.

### Cache Size

Configure via environment variable:
```bash
MEMO_CACHE_SIZE=10240 # Default
MEMO_CACHE_SIZE=8192  # Larger cache for more memoizable steps
```

Monitor cache size if:
- You have many memoizable steps
- Step definitions produce large outputs
- Flow executes with highly variable inputs

## Behavior on Failure

**Only successful executions are cached.**

```
Attempt 1: Step fails
  Cache: miss, execute step
  Result: error
  Cache: NOT updated

Attempt 2: Same inputs
  Cache: miss (failure not cached)
  Execute step again
  Result: success
  Cache: updated with success
```

This prevents caching failures and accidentally skipping retries.

## Example: Currency Conversion

```json
{
  "id": "convert-currency",
  "memoizable": true,
  "type": "sync",
  "http": {
    "endpoint": "https://api.exchangerate-api.com/rates",
    "timeout": 5000
  },
  "attributes": {
    "from_currency": { "required": true },
    "to_currency": { "required": true },
    "amount": { "required": true },
    "converted_amount": { "required": false }
  }
}
```

**Usage:**
```json
{
  "from_currency": "USD",
  "to_currency": "EUR",
  "amount": 100
}
```

**First call:** API called, result cached under hash of `USD:EUR:100`

**Second call with same inputs:** Cache hit, returns EUR equivalent of $100 instantly

**Third call with different amount:** `USD:EUR:200` is different key, API called again

## Monitoring

**To check cache effectiveness:**

1. Look at flow execution times
2. Compare to baseline (without cache)
3. Check step handler logs—if handler is called fewer times than expected, cache is working

**Redis memory impact:**

Cache is in-engine memory, not Redis. Monitor engine process memory usage.

## Limitations

1. **In-memory only**: Cache lost on restart
2. **Per-instance**: Each engine instance has its own cache (not shared)
3. **No persistence**: Cannot export or back up cache entries
4. **No TTL**: No time-based expiration
5. **No manual invalidation**: Cannot manually clear entries (only LRU eviction)

## Interaction with For Each

When a step with `for_each` and `memoizable` executes:

```
items = [apple, banana, apple]

Work item 1 (apple): cache miss, execute, cache result
Work item 2 (banana): cache miss, execute, cache result
Work item 3 (apple): cache hit, return cached result instantly
```

Each work item's inputs are cached independently.

## Troubleshooting

**Q: Step still executing even though I enabled memoizable?**
A: Check:
- Inputs are identical (order matters for the hash)
- Step definition hasn't changed (different code = different key)
- Cache size isn't full (check `MEMO_CACHE_SIZE`)

**Q: Cache got wrong results after I updated the step?**
A: Cache is keyed by step definition. Updating the step creates a new cache key automatically. Old cached values are orphaned.

**Q: Can I share cache across engine instances?**
A: No, cache is in-process. Each instance caches independently. This is fine for redundancy (each instance still gets the benefit locally).

## Best Practices

1. **Use for read-only operations**: Lookups, transformations, API calls that don't change state
2. **Monitor cache hits**: Use logs to verify memoization is effective
3. **Size appropriately**: Don't set `MEMO_CACHE_SIZE` too high (memory waste) or too low (excessive misses)
4. **Update step versions**: When changing logic, update step ID/version to force recompute
5. **Document decisions**: Add notes to step definition explaining why it's memoizable
