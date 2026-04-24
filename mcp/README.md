# Argyll MCP

This MCP server exposes both the Argyll runtime surface and an OpenAPI-driven design surface. It supports step management, flow visibility and control, execution plan previews, health inspection, engine state, and tools for analyzing external REST/JSON services into planner-oriented step registrations. When a service gap cannot be handled by declarative name mapping alone, the MCP can propose a Lua script step to bridge the remaining shape or type mismatch.

## Tools

- `infer_openapi_steps` - infer planner-oriented Argyll step drafts and example plans from an OpenAPI spec, merged with existing registered steps; supports `analyze` and `propose_registrations` modes.
- `analyze_service_spec` - analyze one external REST/JSON service spec and summarize the planner-oriented operations, steps, and plans it exposes.
- `analyze_service_landscape` - analyze multiple service specs together and infer cross-service planning links, missing attributes, and bridge opportunities.
- `propose_bridge_steps` - propose Lua bridge step drafts for missing cross-service planning edges when declarative mapping is not enough.
- `generate_step_impl` - generate a Go or Python SDK implementation draft for a proposed step, including Lua script steps.
- `diff_proposed_steps` - dry-run proposed step registrations against the live catalog and classify each as create, update, or skip; accepts either explicit `steps` or an `infer_openapi_steps` proposal payload.
- `apply_proposed_steps` - apply only the required creates and updates for a proposed step set using the existing registration endpoints; accepts either explicit `steps` or an `infer_openapi_steps` proposal payload.
- `list_steps` - list all registered steps.
- `register_step` - register a new step.
- `get_step` - fetch a single step by ID.
- `update_step` - update a step by ID.
- `unregister_step` - remove a step by ID.
- `preview_plan` - preview an execution plan from goal steps and init state.
- `list_flows` - list flows.
- `query_flows` - query flows by status, labels, ID prefix, sort, and pagination.
- `get_flow` - fetch a single flow by ID.
- `get_flow_status` - fetch the lightweight status payload for a single flow.
- `start_flow` - start a new flow.
- `engine_state` - fetch the current engine state.
- `list_step_health` - list health for all steps.
- `get_step_health` - fetch health for a single step.
- `sdk_step_template` - return a minimal Go or Python SDK step template.

## Resources

- `/sdk/steps` - SDK step implementation guidance.
- `/sdk/go/steps` - Go SDK step patterns.
- `/sdk/python/steps` - Python SDK step patterns.

## Prompts

- `implement_step` - guide an agent through implementing a step with the Go or Python SDK.

## LLM Workflow Example

An LLM can use this MCP to analyze external service specs, infer planner-oriented Argyll step registrations, preview the resulting plans, and then draft bridge implementations only when declarative input/output mapping is not enough.

Example user request to the LLM:

> We have a customer service and an order service. Analyze both OpenAPI specs, figure out what Argyll steps we should register, identify any bridge steps we need, apply the non-redundant registrations, and then draft the Go implementation for any remaining bridge step.

One reasonable tool sequence is:

1. `analyze_service_landscape` Input: both service specs plus any existing registered steps. Output: inferred relationships, missing attributes, and bridge opportunities that still need a Lua transform.
2. `propose_bridge_steps` Input: the landscape result. Output: candidate Lua bridge step definitions.
3. `infer_openapi_steps` Input: each OpenAPI spec with `mode: "propose_registrations"`. Output: spec-backed Argyll step drafts.
4. `diff_proposed_steps` Input: the proposal payload from `infer_openapi_steps`. Output: create/update/skip decisions against the live catalog.
5. `apply_proposed_steps` Input: the same proposal payload. Output: applied registrations.
6. `preview_plan` Input: likely goal steps and initial attrs. Output: whether Argyll can now infer the expected plan.
7. `generate_step_impl` Input: a remaining bridge step proposal. Output: a Go or Python SDK implementation draft, including a script step when needed.

Example LLM instruction:

```text
Use the Argyll MCP to analyze these two OpenAPI specs. First run analyze_service_landscape to understand the cross-service graph. Then use infer_openapi_steps in propose_registrations mode for each spec, diff and apply the non-redundant steps, preview a plan for creating an order from customer_email + items, and if a bridge is still needed, generate a Go or Python script step for it. Prefer declarative input/output mappings first and only use a Lua script when the mapping layer cannot express the reshape. Explain which steps came directly from the OpenAPI docs and which were inferred as script bridges.
```

This is the intended pattern: the LLM should use the MCP to discover the planner graph, register useful steps, verify the resulting plan, and only then generate Lua bridge code where the graph still has gaps.

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

## Example JSON-RPC (stdio framing)

Messages are newline-delimited JSON (one JSON-RPC object per line).

```json
{"jsonrpc":"2.0","id":1,"method":"initialize"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"preview_plan","arguments":{"goals":["step-a","step-b"],"init":{"input":"value"}}}}
```
