# Step Types

Argyll supports four step types. Choose the simplest one that fits your workload.

## Sync HTTP

Use when the work finishes within the HTTP request timeout.

- Engine calls your endpoint
- Your handler returns outputs immediately

## Async HTTP

Use when work is long-running or you need background processing.

- Engine calls your endpoint and includes a webhook URL
- Your handler returns immediately
- You POST the result to the webhook later

## Script

Use Ale or Lua for simple transformations and predicates.

- Runs inside the engine
- No external HTTP server required

## Flow (Sub-flow)

Use when you want reusable sub-workflows.

- Runs a child flow with its own goals
- Inputs and outputs can be mapped between parent and child
