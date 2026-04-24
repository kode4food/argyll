# Argyll SDK Step Implementation Guidance

Use this guidance when implementing Argyll steps with the SDKs.

Core model:
- A step declares required inputs, outputs, execution type, and optional labels, predicates, memoization, retry/work config, and HTTP config.
- Sync and async HTTP steps need an HTTP endpoint. The default invocation method is POST.
- Supported configured HTTP methods are GET, POST, PUT, and DELETE.
- Endpoint placeholders such as /users/{user_id} must correspond to declared input attributes.
- Step HTTP endpoints receive input args directly and return output args directly. Error responses should use HTTP status codes and Problem Details.

SDK-hosted HTTP steps:
- Use Go Start(handler) or Python start(handler) when the SDK should register the step and run the step HTTP server.
- The SDK-hosted server currently handles POST step invocations. Do not generate SDK-hosted GET/PUT/DELETE handlers unless the SDK handler layer is extended first.
- Use WithAsyncExecution or with_async_execution for async steps; the handler returns immediately and completes through the webhook/AsyncContext.

External HTTP steps:
- Use Register/register with WithEndpoint/with_endpoint when another service already implements the HTTP endpoint.
- Use WithMethod/with_method for GET, PUT, or DELETE external endpoints. POST can be omitted because it is the default.

Script and flow steps:
- Use WithScript/with_script plus Register/register for script steps.
- Use WithFlowGoals/with_flow_goals plus Register/register for sub-flow steps.
- For bridge work, prefer declarative input/output name mappings first.
- Use a Lua script step when the mapping layer cannot express the reshape.
