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
k6 run --vus 250 --duration 30s k6-simple.js
```

### Parameters

- `--vus N` - Number of virtual users (default: 1)
- `--duration Xs` - Test duration in seconds (default: infinite)
- `--env ENGINE_URL=http://...` - Engine URL (default: http://localhost:8080)

### Examples

Light load (10 users for 10 seconds):
```bash
k6 run --vus 10 --duration 10s k6-simple.js
```

Medium load (250 users for 30 seconds):
```bash
k6 run --vus 250 --duration 30s k6-simple.js
```

Heavy load (1000 users for 1 minute):
```bash
k6 run --vus 1000 --duration 60s k6-simple.js
```

Against different engine:
```bash
k6 run --vus 100 --duration 30s --env ENGINE_URL=http://staging:8080 k6-simple.js
```

## What It Tests

The test:
1. Registers a simple step that returns `{:result "hello"}`
2. Each VU creates flows with unique IDs
3. Polls for flow completion (max 5 seconds per flow)
4. Tracks success/failure rates and throughput

## Output

The test reports:
- **Duration** - Total test time
- **VUs** - Virtual users used
- **Started** - Flows created
- **Completed** - Flows that finished successfully
- **Failed** - Flows that failed or timed out
- **Throughput** - Flows completed per second
- **Error Rate** - Percentage of failed flows
- **Success Rate** - Percentage of successful flows
