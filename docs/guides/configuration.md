# Configuration and Deployment

This guide covers how to configure Argyll for development, testing, and production deployment.

## Environment Variables

### Engine: Raft Storage

Configure the Raft-backed stores used by engine metadata/state and flow executors:

```bash
RAFT_NODE_ID=argyll-1                         # Local Raft server ID
RAFT_ADDRESS=127.0.0.1:9701                  # Local Raft address
RAFT_DATA_DIR=/tmp/argyll-raft/argyll-1      # Durable local state
RAFT_LOG_TAIL_SIZE=20480                     # Hot retained WAL tail cache entries
RAFT_SERVERS=argyll-1=127.0.0.1:9701         # Bootstrap cluster members
```

### Engine: Runtime + Caching

```bash
API_HOST=0.0.0.0                        # HTTP listen host
API_PORT=8080                           # HTTP API port
WEBHOOK_BASE_URL=http://localhost:8080  # Async callback base URL
LOG_LEVEL=info                          # Log level: debug, info, warn, error
STEP_TIMEOUT=30000                      # Global HTTP step timeout fallback (ms)
TIMEBOX_CACHE_SIZE=4096                 # Shared Timebox projection cache entries
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

## Cluster Topology

Argyll now uses two Raft-backed Timebox stores:

- `EngineStore` for engine metadata/state aggregates such as `catalog`, `nodes`, and `node:<node-id>`
- `FlowStore` for flow state and flow indexes

### Single Node

```bash
RAFT_NODE_ID=argyll-1
RAFT_ADDRESS=127.0.0.1:9701
RAFT_DATA_DIR=/var/lib/argyll/raft/argyll-1
```

### Three Node Cluster

```bash
RAFT_NODE_ID=argyll-1
RAFT_ADDRESS=argyll-1:9701
RAFT_DATA_DIR=/var/lib/argyll/raft/argyll-1
RAFT_SERVERS=argyll-1=argyll-1:9701,argyll-2=argyll-2:9702,argyll-3=argyll-3:9703
```

**Store behaviors:**
- `catalog`, `nodes`, and `node:<node-id>` commit through the engine store's Raft log
- Flow state and flow indexes commit through the flow store
- Timebox indexes live alongside their store and are updated atomically with event appends
- Writes must target the leader node; followers serve reads from their local state

## Development Setup

For local development with Docker Compose:

```bash
docker compose up
# This starts:
# - argyll-engine: localhost:8080
# - argyll-web: localhost:3001
```

Environment variables are already configured in `docker-compose.yml`.

For local testing without Docker:

```bash
# Start a 3-node local cluster
cd engine
./start.sh
```

## Production Setup

### Recommended Configuration

1. **High Availability**: Run 3+ Raft nodes
   - Each node needs a stable `RAFT_NODE_ID`, `RAFT_DATA_DIR`, and `RAFT_ADDRESS`
   - Tune `RAFT_LOG_TAIL_SIZE` only when you need a larger or smaller retained hot log tail; the default is `20480`
   - Keep `RAFT_SERVERS` consistent across the initial voter set
   - Writes commit on quorum, so capacity planning must account for replicated disk I/O

2. **Authentication & Reverse Proxy**:
   - Place engine behind a reverse proxy (nginx, envoy, etc.)
   - Add authentication/authorization at the proxy layer
   - The engine itself has no built-in auth
   - For HAProxy leader-aware write routing, use `GET /health` and inspect the `X-Argyll-Raft-State` response header. It is `leader`, `candidate`, `follower`, or `unknown`, and is derived from constant-time status lookups only.

3. **External Monitoring**:
   - Engine has no built-in Prometheus metrics
   - Integrate with your APM stack (Datadog, New Relic, etc.)
   - Monitor Raft leader changes, disk latency, and follower lag

4. **Logging**:
   - Forward logs to centralized system (ELK, Splunk, etc.)
   - Set `LOG_LEVEL` appropriately (warn/error for prod)

### Performance Tuning

- **Memory**: Engine caches are in-process. Monitor memory growth. Set `MEMO_CACHE_SIZE` based on available memory.
- **Concurrency**: Parallelism is per-step via `work_config` (`parallelism <= 0` means sequential execution with concurrency `1`). No global concurrency limit.
- **Timeout**: `step.http.timeout` overrides per step; otherwise the engine uses `STEP_TIMEOUT` (default `30000` ms)

Write throughput is leader-bound and pays quorum replication plus disk durability cost. Add nodes for availability and operational headroom, not for linear scaling of one write-heavy workload.

### HAProxy Leader-Aware Routing

Use separate backends for reads and writes. Point read traffic at all healthy nodes, and configure the write backend to keep only the Raft leader `UP`.

```haproxy
frontend argyll
  acl write_webhook path_beg /webhook/
  acl write_method method POST PUT DELETE PATCH
  acl write_step path_beg /engine/step
  acl write_flow path -i /engine/flow

  use_backend argyll_write if write_webhook
  use_backend argyll_write if write_method write_step
  use_backend argyll_write if write_method write_flow
  default_backend argyll_read

backend argyll_read
  option httpchk GET /health
  http-check expect status 200
  server n1 argyll-engine-1:8080 check
  server n2 argyll-engine-2:8080 check
  server n3 argyll-engine-3:8080 check

backend argyll_write
  option httpchk GET /health
  http-check expect hdr name "X-Argyll-Raft-State" value -m str leader
  server n1 argyll-engine-1:8080 check
  server n2 argyll-engine-2:8080 check
  server n3 argyll-engine-3:8080 check
```

Keep the default path on `argyll_read`, and only opt known Raft-command routes into `argyll_write`. In the current API surface that means step mutations, `POST /engine/flow`, and webhook callbacks. Read-style POST endpoints such as `POST /engine/plan` and `POST /engine/flow/query` should stay on `argyll_read`.

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
2. **Raft health**: Leader stability, election churn, quorum status
3. **Store health**: Storage latency, replication pressure, free space
4. **Flow completion**: Success rate, latency distribution
5. **Step execution**: Per-step failure rate, p95 latency

### Logs to Watch

```
ERROR: failed to create raft store   # Shared store startup issue
ERROR: step execution failed         # Individual step failure
WARN: leadership lost                # Quorum or shutdown issue
```

### Recommended Tools

- **Metrics**: Prometheus + Grafana (with custom instrumentation)
- **Logs**: ELK Stack, Splunk, or CloudWatch
- **APM**: Datadog, New Relic, or Jaeger (with custom tracing)

Since Argyll has no built-in metrics, instrument at:
- Reverse proxy layer (request count, latency, errors)
- Step handler layer (execution time, error rate)
- Raft and store layer (leader changes, storage latency, queueing)

## Troubleshooting

### Engine Won't Start

```
Error: failed to create raft store
→ Check RAFT_ADDRESS / RAFT_SERVERS
→ Verify RAFT_DATA_DIR exists and is writable
→ Check peer reachability and local port conflicts
```

### Flows Stuck in Active State

```
→ Check logs for step failures or leadership churn
→ Verify step handlers are reachable
→ Check quorum health and follower connectivity
```

### High Memory Usage

```
→ Check MEMO_CACHE_SIZE and TIMEBOX_CACHE_SIZE
→ Look for flows with large aggregated outputs
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
