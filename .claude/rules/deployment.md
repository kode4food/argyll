# Deployment & Configuration

## Environment Variables

```bash
# Engine store (steps, health checks)
ENGINE_REDIS_ADDR=localhost:6379
ENGINE_REDIS_PASSWORD=
ENGINE_REDIS_DB=0
ENGINE_REDIS_PREFIX=argyll:engine

# Flow store (flow execution state)
FLOW_REDIS_ADDR=localhost:6379
FLOW_REDIS_PASSWORD=
FLOW_REDIS_DB=0
FLOW_REDIS_PREFIX=argyll:flow

# Archiving policy (external archiver)
ARCHIVE_MEMORY_PERCENT=80
ARCHIVE_MAX_AGE=24h
ARCHIVE_MEMORY_CHECK_INTERVAL=5s
ARCHIVE_SWEEP_INTERVAL=1h
ARCHIVE_LEASE_TIMEOUT=15m
ARCHIVE_PRESSURE_BATCH=10
ARCHIVE_SWEEP_BATCH=100

# Archiver backend (argyll-s3)
ARCHIVE_BUCKET_URL=s3://bucket-name
ARCHIVE_PREFIX=archived/
ARCHIVE_POLL_INTERVAL=500ms
```

Archiving runs in the `argyll-s3` service.

## Store Separation

Engine and flow stores can use different Valkey instances:

- **Single instance**: Both stores point to same Valkey (default)
- **Separated concerns**: Engine state on one instance, flows on another
- **Flow sharding**: Multiple Valkey instances for partitioning

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
2. Use dedicated Valkey instances for engine vs flow state
3. Limit who can create script steps
4. Add external monitoring (no built-in metrics)
5. Configure archiving and an external consumer for long-term deactivated flow storage

**Performance:**
- Peak throughput: 6000+ flows/second per instance
- Linear scaling with additional instances
- Event sourcing overhead minimal at scale

## Step Retry Configuration

- **Retryable Errors**: Network failures, timeouts, 5xx HTTP errors
- **Permanent Errors**: 200 OK with `success: false` - no retry
- **Backoff Strategies**: Fixed, linear, or exponential
- **Per-Step Override**: Steps can override global retry config
