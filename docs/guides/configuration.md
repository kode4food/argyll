# Configuration and Deployment

This guide covers how to configure Argyll for development, testing, and production deployment.

## Environment Variables

### Engine Storage

Configure where the engine stores step definitions and health data:

```bash
ENGINE_REDIS_ADDR=localhost:6379       # Default
ENGINE_REDIS_PASSWORD=                 # Empty if no auth
ENGINE_REDIS_DB=0                      # Redis database number
ENGINE_REDIS_PREFIX=argyll:engine       # Namespace prefix
```

### Flow Storage

Configure where flows and their state are stored:

```bash
FLOW_REDIS_ADDR=localhost:6379
FLOW_REDIS_PASSWORD=
FLOW_REDIS_DB=0
FLOW_REDIS_PREFIX=argyll:flow
```

### Archiving Policy

Configure when and how flows are archived:

```bash
ARCHIVE_MEMORY_PERCENT=80              # Trigger archiving when Redis reaches 80% full
ARCHIVE_MAX_AGE=24h                    # Archive flows older than 24 hours
ARCHIVE_MEMORY_CHECK_INTERVAL=5s       # Check memory pressure every 5 seconds
ARCHIVE_SWEEP_INTERVAL=1h              # Run archiving sweep every hour
ARCHIVE_LEASE_TIMEOUT=15m              # Lease duration for archive jobs
ARCHIVE_PRESSURE_BATCH=10              # Archive 10 flows per pressure event
ARCHIVE_SWEEP_BATCH=100                # Archive 100 flows per sweep
```

### Archiver Backend (S3)

```bash
ARCHIVE_BUCKET_URL=s3://my-bucket      # S3 bucket URL
ARCHIVE_PREFIX=archived/                # Prefix for archived objects
ARCHIVE_POLL_INTERVAL=500ms             # Poll interval for archive job status
```

### Caching

```bash
MEMO_CACHE_SIZE=4096                   # Max memoization cache entries (default: 4096)
```

### Server Configuration

```bash
ENGINE_PORT=8080                       # HTTP API port
ENGINE_LOGGING_LEVEL=info              # Log level: debug, info, warn, error
```

## Store Separation

Engine and flow stores can use different Valkey instances. This is useful for scaling:

### Single Instance (Default)

Both stores point to the same Valkey:

```bash
ENGINE_REDIS_ADDR=valkey:6379
FLOW_REDIS_ADDR=valkey:6379
```

### Separated Stores

Engine state on one instance, flows on another:

```bash
ENGINE_REDIS_ADDR=valkey-engine:6379
FLOW_REDIS_ADDR=valkey-flows:6379
```

**Benefits:**
- Scale engine and flow stores independently
- Isolate workloads
- Easier debugging and monitoring

### Flow Sharding

Multiple Valkey instances for flow partitioning:

```bash
# In a more advanced setup, you could:
# - Use Redis Cluster for automatic sharding
# - Or deploy multiple engines with different FLOW_REDIS_ADDR
```

## Development Setup

For local development with Docker Compose:

```bash
docker compose up
# This starts:
# - valkey (Redis): localhost:6379
# - argyll-engine: localhost:8080
# - argyll-web: localhost:3001
```

Environment variables are already configured in `docker-compose.yml`.

For local testing without Docker:

```bash
# Start Redis
redis-server

# Set minimal env vars
export ENGINE_REDIS_ADDR=localhost:6379
export FLOW_REDIS_ADDR=localhost:6379
export ENGINE_REDIS_PREFIX=argyll:engine:test
export FLOW_REDIS_PREFIX=argyll:flow:test

# Run engine
go run ./cmd/argyll
```

## Production Setup

### Recommended Configuration

1. **High Availability**: Run 2+ engine instances
   - All instances consume from the same event stream
   - Optimistic concurrency prevents duplicates
   - Natural load balancing

2. **Separate Stores**: Use different Valkey instances for engine vs flow state

3. **Authentication & Reverse Proxy**:
   - Place engine behind a reverse proxy (nginx, envoy, etc.)
   - Add authentication/authorization at the proxy layer
   - The engine itself has no built-in auth

4. **External Monitoring**:
   - Engine has no built-in Prometheus metrics
   - Integrate with your APM stack (Datadog, New Relic, etc.)
   - Monitor Redis memory and latency

5. **Archiving**:
   - Configure archiving policy
   - Set up S3 or compatible backend
   - Monitor archive job success/failure

6. **Logging**:
   - Forward logs to centralized system (ELK, Splunk, etc.)
   - Set `ENGINE_LOGGING_LEVEL` appropriately (warn/error for prod)

### Performance Tuning

- **Memory**: Engine caches are in-process. Monitor memory growth. Set `MEMO_CACHE_SIZE` based on available memory.
- **Concurrency**: Parallelism is per-step via `work_config`. No global concurrency limit.
- **Timeout**: HTTP step timeout is per-step (see [Step Types](../concepts/steps.md))

**Peak throughput:** 6000+ flows/second per engine instance (benchmark-dependent)

**Scaling:** Linear scaling with additional instances. Add capacity by running more engines.

## Security Considerations

### Script Execution

Scripts (Ale and Lua) run inside the engine with restricted capabilities:

**Ale:**
- Purely functional, no I/O
- No resource limits
- Safe for untrusted scripts

**Lua:**
- Partial sandboxing (io, os, debug modules excluded)
- No resource limits
- Use only for trusted scripts

**Recommendation:** Only allow trusted users to create script steps.

### Input Validation

- **UI**: Validates inputs before sending to engine
- **Server**: Validates that all required inputs are present (as defined in the execution plan). Extraneous inputs are accepted but ignored.
- **No type validation**: Server doesn't validate input types against step definitions

**Implication:** Required inputs must be provided to start a flow. Optional inputs can be omitted (steps use defaults). Validate input types and semantics in your step handlers.

### Authentication & Authorization

The engine has no built-in authentication. Options:

1. **Reverse Proxy**: Add auth at the proxy layer (nginx, envoy)
2. **Network Isolation**: Run engine in private network, access via VPN
3. **Mutual TLS**: Use mTLS for service-to-service communication

## Step Retry Configuration

Per-step retry behavior:

**Retryable Errors:**
- Network failures (connection refused, timeout)
- HTTP 5xx errors (500, 502, 503, etc.)

**Permanent Errors:**
- 200 OK with `success: false` - counts as handled error, no retry
- 4xx errors (typically) - no retry

**Backoff Strategies:**

Configure via `work_config`:

```json
{
  "work_config": {
    "max_retries": 3,
    "backoff_ms": 100,
    "max_backoff_ms": 5000,
    "backoff_type": "exponential"
  }
}
```

**Backoff Types:**
- `fixed`: Same delay between retries (backoff_ms)
- `linear`: Delay increases linearly (attempt * backoff_ms)
- `exponential`: Delay doubles each retry (2^attempt * backoff_ms, capped at max_backoff_ms)

**Example:**
```json
{
  "max_retries": 3,
  "backoff_ms": 100,
  "max_backoff_ms": 5000,
  "backoff_type": "exponential"
}
```

Retry delays: 100ms, 200ms, 400ms (capped at 5000ms)

## Health Checks

HTTP steps can include a health check endpoint:

```json
{
  "type": "sync",
  "http": {
    "endpoint": "https://api.example.com/process",
    "health_check": "https://api.example.com/health"
  }
}
```

Health checks run periodically to detect step availability. If unavailable, steps using this handler fail.

## Monitoring & Observability

### What to Monitor

1. **Engine health**: Is the engine process running?
2. **Redis health**: Latency, memory usage, replication lag
3. **Flow completion**: Success rate, latency distribution
4. **Step execution**: Per-step failure rate, p95 latency
5. **Archive jobs**: Success/failure, processed flows per hour

### Logs to Watch

```
ERROR: step execution failed         # Individual step failure
ERROR: flow failed                   # Goal step failed, flow is terminal
WARN: redis connection lost          # Store connectivity issue
WARN: archive job failed             # Flow archiving error
```

### Recommended Tools

- **Metrics**: Prometheus + Grafana (with custom instrumentation)
- **Logs**: ELK Stack, Splunk, or CloudWatch
- **APM**: Datadog, New Relic, or Jaeger (with custom tracing)

Since Argyll has no built-in metrics, instrument at:
- Reverse proxy layer (request count, latency, errors)
- Step handler layer (execution time, error rate)
- Redis layer (latency, memory, key count)

## Troubleshooting

### Engine Won't Start

```
Error: redis: connection refused
→ Check REDIS_ADDR is correct
→ Verify Redis is running
→ Check network connectivity
```

### Flows Stuck in Active State

```
→ Check logs for step failures
→ Verify step handlers are reachable
→ Check Redis memory (may be full, blocking operations)
```

### High Memory Usage

```
→ Check MEMO_CACHE_SIZE setting
→ Look for flows with large aggregated outputs
→ Monitor archive job effectiveness
```

### Step Timeouts

```
→ Increase http.timeout for slow handlers
→ Check downstream system latency
→ Consider async steps for long-running work
```

## Upgrading

Argyll uses event sourcing, so upgrades are generally safe:

1. **No breaking schema changes**: Event format is stable
2. **Compatible updates**: Engine versions can coexist briefly (rolling updates)
3. **Replay safety**: Old flows can be replayed with new versions

**Best practice:**
1. Update one engine instance
2. Monitor for issues
3. Update remaining instances

In-flight flows will complete on whichever instance processes them next.
