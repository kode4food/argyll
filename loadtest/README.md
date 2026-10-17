# Argyll Load Testing

Simple k6 load test for Argyll orchestrator.

## Prerequisites

Install k6:
```bash
brew install k6
```

## Running the Test

Basic usage:
```bash
k6 run --vus 250 --duration 30s k6-start.js
```

### Parameters

- `--vus N` - Number of virtual users (default: 1)
- `--duration Xs` - Test duration in seconds (default: infinite)
- `--env ENGINE_URL=http://...` - Engine URL to target (default: `http://localhost:8080`)

`ENGINE_URL` must point at the Raft leader node in the cluster.

### Examples

Light load (10 users for 10 seconds):
```bash
k6 run --vus 10 --duration 10s k6-start.js
```

Medium load (250 users for 30 seconds):
```bash
k6 run --vus 250 --duration 30s k6-start.js
```

Heavy load (1000 users for 1 minute):
```bash
k6 run --vus 1000 --duration 60s k6-start.js
```

Against different engine:
```bash
k6 run --vus 100 --duration 30s --env ENGINE_URL=http://staging:8080 k6-start.js
```

Against a local 3-node Raft cluster:
```bash
k6 run --vus 250 --duration 30s --env ENGINE_URL=http://localhost:8081 k6-start.js
```

## What It Tests

The test:
1. Registers a Lua script step that returns `{result = "hello"}`
2. Each VU starts flows with unique IDs
3. Tracks success/failure rates and throughput

A script step runs in the transaction that starts its flow, so a flow has already completed when its start request returns.

## Output

The test reports:
- **Duration** - Total test time
- **VUs** - Virtual users used
- **Started** - Flows started and completed
- **Failed** - Start requests that failed
- **Throughput** - Flows per second
- **Error Rate** - Percentage of failed requests
- **Success Rate** - Percentage of successful requests
