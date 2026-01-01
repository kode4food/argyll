# Deployment & Configuration

## Environment Variables

```bash
# Engine store (steps, health checks)
ENGINE_REDIS_ADDR=localhost:6379
ENGINE_REDIS_PASSWORD=
ENGINE_REDIS_DB=0
ENGINE_REDIS_PREFIX=argyll:engine

# Workflow store (workflow execution state)
WORKFLOW_REDIS_ADDR=localhost:6379
WORKFLOW_REDIS_PASSWORD=
WORKFLOW_REDIS_DB=0
WORKFLOW_REDIS_PREFIX=argyll:workflow

# Hibernation (optional - archives completed flows to blob storage)
HIBERNATOR_URL=s3://bucket-name?region=us-east-1  # or gs://, azblob://
HIBERNATOR_PREFIX=archived/
```

## Store Separation

Engine and workflow stores can use different Valkey instances:

- **Single instance**: Both stores point to same Valkey (default)
- **Separated concerns**: Engine state on one instance, workflows on another
- **Workflow sharding**: Multiple Valkey instances for partitioning

## Security Considerations

**Script Execution**
- Ale: Purely functional, no I/O capabilities, no resource limits
- Lua: Partial sandboxing (io/os/debug excluded), no resource limits
- Recommendation: Only allow trusted users to create script steps

**Input Validation**
- Server doesn't strictly validate inputs against execution plan
- Extraneous inputs accepted but ignored
- No type validation on input values

## Production Setup

**Recommended:**
1. Run 2+ engine instances for high availability
2. Use dedicated Valkey instances for engine vs workflow state
3. Limit who can create script steps
4. Add external monitoring (no built-in metrics)
5. Configure hibernation for long-term flow archival

**Performance:**
- Peak throughput: 6000+ workflows/second per instance
- Linear scaling with additional instances
- Event sourcing overhead minimal at scale

## Step Retry Configuration

- **Retryable Errors**: Network failures, timeouts, 5xx HTTP errors
- **Permanent Errors**: 200 OK with `success: false` - no retry
- **Backoff Strategies**: Fixed, linear, or exponential
- **Per-Step Override**: Steps can override global retry config
