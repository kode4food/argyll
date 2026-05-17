# Argyll MCP

This MCP server exposes both the Argyll runtime surface and an OpenAPI-driven design surface. It supports step management, flow visibility and control, execution plan previews, health inspection, engine state, and tools for analyzing external REST/JSON services into planner-oriented step registrations.

## Tools

- `analyze_openapi_contract` - extract neutral REST contract facts from an OpenAPI spec for LLM-driven Argyll registration design.
- `analyze_service_spec` - analyze one external REST/JSON service spec and summarize the operations it exposes.
- `analyze_service_landscape` - analyze multiple service specs together and describe cross-service planning links and missing attributes.
- `generate_step_impl` - generate a Go or Python SDK implementation draft for a proposed step.
- `diff_proposed_steps` - dry-run proposed step registrations against the live catalog and classify each as create, update, or skip.
- `apply_proposed_steps` - apply only the required creates and updates for a proposed step set using the existing registration endpoints.
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
- `/openapi/ingestion` - OpenAPI ingestion guidance for LLM-authored Argyll registrations.

## Prompts

- `implement_step` - guide an agent through implementing a step with the Go or Python SDK.
- `ingest_openapi_services` - guide an agent through OpenAPI contract analysis, registration authoring, readback verification, and plan preview.

## LLM Workflow Example

An LLM can use this MCP to analyze external service specs, author planner-oriented Argyll step registrations, preview the resulting plans, and surface any remaining mapping gaps for the user to resolve.

`analyze_openapi_contract` includes an `argyll_capabilities` block describing declarative step features the LLM can rely on. This includes `required.mapping`, `optional.mapping`, `output.mapping`, and `required.match`. The tool reports neutral contract facts; the LLM authors planner-facing Argyll step definitions and uses role-specific mapping config instead of creating a synthetic bridge step.

The same capability block also advertises endpoint arguments. Argyll replaces `{name}` placeholders in HTTP step endpoints with runtime args, so OpenAPI path and query parameters are represented as endpoint placeholders such as `/customers/{customerId}?type={type}`.

Example user request to the LLM:

> We have a customer service and an order service. Analyze both OpenAPI specs, figure out what Argyll steps we should register, apply the non-redundant registrations, and preview whether Argyll can plan an order flow from the initial customer email and items.

One reasonable tool sequence is:

1. `analyze_service_landscape` Input: both service specs plus any existing registered steps. Output: discovered relationships and missing attributes.
2. `analyze_openapi_contract` Input: each OpenAPI spec. Output: neutral service facts and Argyll capability guidance.
3. LLM-authored step definitions: canonical planner attributes, role-specific mappings, and required matches where the service contract and business flow justify them.
4. `diff_proposed_steps` Input: the authored steps. Output: create/update/skip decisions against the live catalog.
5. `apply_proposed_steps` Input: the authored steps. Output: applied registrations plus readback verification.
6. `preview_plan` Input: likely goal steps and initial attrs. Output: whether Argyll can now build the expected plan.
7. `generate_step_impl` Input: a proposed SDK-hosted step. Output: a Go or Python SDK implementation draft.

Example LLM instruction:

```text
Use the Argyll MCP to analyze these two OpenAPI specs. First run analyze_service_landscape to understand the cross-service graph. Then use analyze_openapi_contract for each spec, author planner-facing Argyll step definitions from the contract facts, diff and apply the non-redundant steps, and preview a plan for creating an order from customer_email + items. Explain which steps came directly from the OpenAPI docs and call out any remaining missing attributes.
```

This is the intended pattern: the LLM should use the MCP to discover the planner graph, register useful steps, and verify the resulting plan. If the graph still has gaps, the MCP reports them instead of fabricating mapping steps.

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
