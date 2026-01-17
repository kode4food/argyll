# Argyll MCP

This MCP server exposes a small interface over the Argyll engine. It supports
listing steps, fetching a single step, previewing execution plans, and fetching
engine state.

## Tools

- `list_steps` - list all registered steps
- `get_step` - fetch a single step by ID
- `preview_plan` - preview an execution plan from goal steps and init state
- `engine_state` - fetch the current engine state

## Run

```bash
make run
```

Override the engine URL:

```bash
./dist/argyll-mcp -engine http://localhost:8080
```

## Build

```bash
make build
```

## Example JSON-RPC

```json
{"jsonrpc":"2.0","id":1,"method":"initialize"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"preview_plan","arguments":{"goals":["step-a","step-b"],"init":{"input":"value"}}}}
```
