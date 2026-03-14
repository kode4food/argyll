# Configuration and Deployment

This guide covers how to configure Argyll for development, testing, and production deployment.

## Environment Variables

### Engine: Catalog Storage

Configure where the engine stores the step catalog:

```bash
CATALOG_POSTGRES_URL='postgres://localhost:5432/argyll?sslmode=disable'
CATALOG_POSTGRES_PREFIX=argyll
CATALOG_POSTGRES_MAX_CONNS=96
CATALOG_SNAPSHOT_WORKERS=4
```

### Engine: Partition + Flow Storage

Configure where the engine stores partition state and flow state:

```bash
PARTITION_POSTGRES_URL='postgres://localhost:5432/argyll?sslmode=disable'
PARTITION_POSTGRES_PREFIX=argyll
PARTITION_POSTGRES_MAX_CONNS=96
PARTITION_SNAPSHOT_WORKERS=4

FLOW_POSTGRES_URL='postgres://localhost:5432/argyll?sslmode=disable'
FLOW_POSTGRES_PREFIX=argyll
FLOW_POSTGRES_MAX_CONNS=96
FLOW_CACHE_SIZE=4096
```

### Engine: Runtime + Caching

```bash
API_HOST=0.0.0.0                        # HTTP listen host
API_PORT=8080                           # HTTP API port
WEBHOOK_BASE_URL=http://localhost:8080  # Async callback base URL
LOG_LEVEL=info                          # Log level: debug, info, warn, error
STEP_TIMEOUT=30000                      # Global HTTP step timeout fallback (ms)
FLOW_CACHE_SIZE=4096                    # Flow aggregate cache entries
MEMO_CACHE_SIZE=10240                   # Memoization cache entries
```

HTTP step timeout is set per step via `step.http.timeout` (milliseconds). If omitted/<=0, the engine uses `STEP_TIMEOUT` (default: `30000` ms).

### Engine: Retry Defaults

These values are used when a step omits retry settings, or sets retry fields to zero/empty values (for example no `work_config`, or `work_config` only setting non-retry fields like `parallelism`):

```bash
RETRY_MAX_RETRIES=10                    # Default max retries
RETRY_INITIAL_BACKOFF=1000              # Initial backoff in milliseconds
RETRY_MAX_BACKOFF=60000                 # Backoff cap in milliseconds
RETRY_BACKOFF_TYPE=exponential          # fixed, linear, exponential
```

These defaults must be valid at startup:
- `RETRY_MAX_RETRIES` cannot be `0`
- `RETRY_INITIAL_BACKOFF` must be `> 0`
- `RETRY_MAX_BACKOFF` must be `> 0` and `>= RETRY_INITIAL_BACKOFF`
- `RETRY_BACKOFF_TYPE` must be `fixed`, `linear`, or `exponential`

### Archiver: Policy

The external archiver package has not been migrated to Postgres yet. The engine itself can persist archive records through `timebox/postgres`, but the standalone Argyll archiver still documents the older Redis-based flow.

```bash
ARCHIVE_MEMORY_PERCENT=80              # Trigger archiving when the archive store reaches 80% full
ARCHIVE_MAX_AGE=24h                    # Archive flows older than 24 hours
ARCHIVE_MEMORY_CHECK_INTERVAL=5s       # Check memory pressure every 5 seconds
ARCHIVE_POLL_INTERVAL=500ms            # Poll interval for archive stream consumption
ARCHIVE_SWEEP_INTERVAL=1h              # Run archiving sweep every hour
ARCHIVE_LEASE_TIMEOUT=15m              # Lease duration for archive jobs
ARCHIVE_PRESSURE_BATCH=10              # Archive 10 flows per pressure event
ARCHIVE_SWEEP_BATCH=100                # Archive 100 flows per sweep
```

### Archiver: Backend (Bucket)

```bash
ARCHIVE_BUCKET_URL=s3://my-bucket      # Bucket URL
ARCHIVE_PREFIX=archived/               # Prefix for archived objects
```

### Archiver: Backend (File Sink)

```bash
ARCHIVE_SINK_PATH=/dev/null             # Local filesystem sink path
```

## Store Separation

Catalog, Partition, and Flow stores are configured independently. In the default local setup they all point at the same Postgres database and prefix, and Argyll shares one underlying Postgres persistence across them.

### Single Instance (Default)

```bash
CATALOG_POSTGRES_URL='postgres://postgres:postgres@postgres:5432/argyll?sslmode=disable'
PARTITION_POSTGRES_URL="$CATALOG_POSTGRES_URL"
FLOW_POSTGRES_URL="$CATALOG_POSTGRES_URL"
```

### Separated Catalog

Catalog on its own database; partition and flow on another:

```bash
CATALOG_POSTGRES_URL='postgres://catalog-db:5432/argyll?sslmode=disable'
PARTITION_POSTGRES_URL='postgres://flow-db:5432/argyll?sslmode=disable'
FLOW_POSTGRES_URL="$PARTITION_POSTGRES_URL"
```

**Store behaviors:**
- Catalog: event trimming enabled, snapshotted on shutdown
- Partition: event trimming enabled, snapshotted on shutdown
- Flow: full event history retained (no trimming)

## Development Setup

For local development with Docker Compose:

```bash
docker compose up
# This starts:
# - postgres: localhost:5432
# - argyll-engine: localhost:8080
# - argyll-web: localhost:3001
```

Environment variables are already configured in `docker-compose.yml`.

For local testing without Docker:

```bash
# Start Postgres
docker compose up postgres

# Set minimal env vars
export CATALOG_POSTGRES_URL='postgres://localhost:5432/argyll?sslmode=disable'
export PARTITION_POSTGRES_URL="$CATALOG_POSTGRES_URL"
export FLOW_POSTGRES_URL="$CATALOG_POSTGRES_URL"
export CATALOG_POSTGRES_PREFIX=argyll
export PARTITION_POSTGRES_PREFIX=argyll
export FLOW_POSTGRES_PREFIX=argyll

# Run engine
go run ./cmd/argyll
```

## Production Setup

### Recommended Configuration

1. **High Availability**: Run 2+ engine instances
   - All instances consume from the same event stream
   - Optimistic concurrency prevents duplicates
   - Natural load balancing

2. **Separate Catalog vs Partition/Flow Stores**:
   - Keep catalog on its own Postgres database or schema if you need isolation
   - Flow state can share the same backend or use a dedicated one

3. **Authentication & Reverse Proxy**:
   - Place engine behind a reverse proxy (nginx, envoy, etc.)
   - Add authentication/authorization at the proxy layer
   - The engine itself has no built-in auth

4. **External Monitoring**:
   - Engine has no built-in Prometheus metrics
   - Integrate with your APM stack (Datadog, New Relic, etc.)
   - Monitor Postgres connections, latency, and WAL pressure

5. **Archiving**:
   - Configure archiving policy
   - Set up S3 or compatible backend
   - Run one archiver process per partition store (`PARTITION_*`)
   - Monitor archive job success/failure

6. **Logging**:
   - Forward logs to centralized system (ELK, Splunk, etc.)
   - Set `LOG_LEVEL` appropriately (warn/error for prod)

### Performance Tuning

- **Memory**: Engine caches are in-process. Monitor memory growth. Set `MEMO_CACHE_SIZE` based on available memory.
- **Concurrency**: Parallelism is per-step via `work_config` (`parallelism <= 0` means sequential execution with concurrency `1`). No global concurrency limit.
- **Timeout**: `step.http.timeout` overrides per step; otherwise the engine uses `STEP_TIMEOUT` (default `30000` ms)

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

**Implication:** Required inputs must be provided to start a flow. Optional inputs can be omitted; defaults are only applied when explicitly declared on the attribute. Validate input types and semantics in your step handlers.

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
    "init_backoff": 100,
    "max_backoff": 5000,
    "backoff_type": "exponential"
  }
}
```

Step-level retry fields only override when they are non-zero/non-empty. If a retry field is omitted or set to zero/empty, the engine uses the global retry default for that field.

**Backoff Types:**
- `fixed`: Same delay between retries (`init_backoff` milliseconds)
- `linear`: Delay increases linearly (attempt * `init_backoff` milliseconds)
- `exponential`: Delay doubles each retry (2^attempt * `init_backoff` milliseconds, capped at `max_backoff`)

**Example:**
```json
{
  "max_retries": 3,
  "init_backoff": 100,
  "max_backoff": 5000,
  "backoff_type": "exponential"
}
```

Retry delays: 100ms, 200ms, 400ms (capped at 5000ms)

## Health Checks

HTTP steps can include a health check endpoint:

```json
{
  "id": "process-payload",
  "name": "Process Payload",
  "type": "sync",
  "http": {
    "endpoint": "https://api.example.com/process",
    "health_check": "https://api.example.com/health"
  },
  "attributes": {
    "payload": { "role": "required", "type": "object" },
    "processed": { "role": "output", "type": "object" }
  }
}
```

Health checks run periodically to update step health status. They do not directly block or fail step execution.

## Monitoring & Observability

### What to Monitor

1. **Engine health**: Is the engine process running?
2. **Postgres health**: Latency, connections, WAL pressure
3. **Flow completion**: Success rate, latency distribution
4. **Step execution**: Per-step failure rate, p95 latency
5. **Archive jobs**: Success/failure, processed flows per hour

### Logs to Watch

```
ERROR: step execution failed         # Individual step failure
ERROR: flow failed                   # Goal step failed, flow is terminal
WARN: store connection lost          # Store connectivity issue
WARN: archive job failed             # Flow archiving error
```

### Recommended Tools

- **Metrics**: Prometheus + Grafana (with custom instrumentation)
- **Logs**: ELK Stack, Splunk, or CloudWatch
- **APM**: Datadog, New Relic, or Jaeger (with custom tracing)

Since Argyll has no built-in metrics, instrument at:
- Reverse proxy layer (request count, latency, errors)
- Step handler layer (execution time, error rate)
- Postgres layer (latency, connections, WAL, table bloat)

## Troubleshooting

### Engine Won't Start

```
Error: connection refused
→ Check CATALOG_POSTGRES_URL / PARTITION_POSTGRES_URL / FLOW_POSTGRES_URL
→ Verify Postgres is running
→ Check network connectivity
```

### Flows Stuck in Active State

```
→ Check logs for step failures
→ Verify step handlers are reachable
→ Check Postgres connections and lock contention
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
