# Spuds API Documentation

This directory contains OpenAPI 3.0 specifications for the Spuds workflow engine:

- **`engine-api.yaml`**: REST API for the Spuds Engine (for engine consumers)
- **`step-interface.yaml`**: Contract specification for step implementors

---

## Engine API (`engine-api.yaml`)

### Audience
Application developers who want to:
- Register and manage workflow steps
- Create and execute workflows
- Monitor workflow execution
- Integrate Spuds into their systems

### Key Endpoints

**Step Management:**
- `POST /engine/step` - Register a new step
- `GET /engine/step` - List all registered steps
- `GET /engine/step/{stepId}` - Get step details
- `PUT /engine/step/{stepId}` - Update step registration
- `DELETE /engine/step/{stepId}` - Unregister step

**Workflow Management:**
- `POST /engine/workflow` - Create and start a workflow
- `GET /engine/workflow` - List all workflows
- `GET /engine/workflow/{workflowId}` - Get workflow state
- `POST /engine/plan` - Preview execution plan (without starting)

**Monitoring:**
- `GET /health` - Engine health check
- `GET /engine` - Complete engine state
- `GET /engine/health` - Step health status
- `GET /engine/ws` - WebSocket for real-time events

**Webhooks:**
- `POST /webhook/{workflowId}/{stepId}/{token}` - Async step completion callback

### Getting Started

1. **Register a step:**
```bash
curl -X POST http://localhost:8080/engine/step \
  -H "Content-Type: application/json" \
  -d '{
    "id": "text-processor",
    "name": "Text Processor",
    "type": "sync",
    "attributes": {
      "input_text": {
        "role": "required",
        "type": "string"
      },
      "processed_text": {
        "role": "output",
        "type": "string"
      }
    },
    "version": "1.0.0",
    "http": {
      "endpoint": "http://localhost:8081/process-text",
      "health_check": "http://localhost:8081/health",
      "timeout": 30000
    }
  }'
```

2. **Start a workflow:**
```bash
curl -X POST http://localhost:8080/engine/workflow \
  -H "Content-Type: application/json" \
  -d '{
    "id": "wf-001",
    "goal_steps": ["text-processor"],
    "initial_state": {
      "input_text": "Hello, Spuds!"
    }
  }'
```

3. **Monitor workflow:**
```bash
curl http://localhost:8080/engine/workflow/wf-001
```

---

## Step Interface (`step-interface.yaml`)

### Audience
Developers implementing workflow steps who need to understand:
- The HTTP contract their endpoints must implement
- Request/response formats
- Error handling patterns
- Deployment options

### Overview

The Step interface defines a standard HTTP contract that enables any endpoint to participate in Spuds workflows. This flexible design allows:

- **Dedicated Services**: Microservices with a single step endpoint
- **Multi-Step Services**: Larger services exposing multiple step endpoints
- **Function-as-a-Service**: Cloud functions, AWS Lambda, etc.
- **Legacy System Integration**: Existing APIs wrapped to conform to the Step interface

## Interface Contract

### Endpoint Pattern
```
POST /{step-endpoint}
```

Where `{step-endpoint}` is any path that implements the Step interface.

### Request Format
```json
{
  "arguments": {
    "required_arg1": "value1",
    "optional_arg2": "value2"
  },
  "metadata": {
    "workflow_id": "wf-123",
    "step_id": "unique-step-identifier",
    "webhook_url": "http://localhost:8080/webhook/wf-123/unique-step-identifier/tok_abc123"
  }
}
```

**Note:** The `metadata` field contains:
- `workflow_id` - The executing workflow's ID
- `step_id` - The step identifier being executed
- `webhook_url` - For async steps only, where to POST completion results

**Idempotency:** Use `workflow_id` + `step_id` as a composite key to ensure each step execution is processed only once.

### Response Format
```json
{
  "success": true,
  "outputs": {
    "result1": "output_value1",
    "result2": "output_value2"
  },
  "is_terminal_stop": false
}
```

### Error Handling
```json
{
  "success": false,
  "error": "Business logic error message"
}
```

### Exception Handling
```json
{
  "success": false,
  "exception": "Fatal error that should terminate the workflow"
}
```

## Step Registration

Steps are registered with the Spuds engine by providing:

1. **Unique ID**: Identifies this step type
2. **Type**: `sync`, `async`, or `script`
3. **Attributes**: Required inputs, optional inputs, and outputs with types
4. **Configuration**: HTTP endpoint (for sync/async) or script code (for script steps)

Example registration (sync HTTP step):
```json
{
  "id": "text-processor",
  "name": "Text Processing Step",
  "type": "sync",
  "attributes": {
    "input_text": {
      "role": "required",
      "type": "string"
    },
    "format": {
      "role": "optional",
      "type": "string",
      "default": "\"uppercase\""
    },
    "processed_text": {
      "role": "output",
      "type": "string"
    }
  },
  "version": "1.0.0",
  "http": {
    "endpoint": "https://myservice.com/api/v1/process-text",
    "health_check": "https://myservice.com/health",
    "timeout": 30000
  }
}
```

## Implementation Examples

### Simple Function
```javascript
// AWS Lambda or similar
exports.handler = async (event) => {
  const { arguments: args } = JSON.parse(event.body);

  if (!args.input_text) {
    return {
      statusCode: 200,
      body: JSON.stringify({
        success: false,
        error: "Missing required argument: input_text"
      })
    };
  }

  return {
    statusCode: 200,
    body: JSON.stringify({
      success: true,
      outputs: {
        processed_text: args.input_text.toUpperCase(),
        processing_time_ms: 42
      }
    })
  };
};
```

### Express.js Service
```javascript
const express = require('express');
const app = express();

// Single dedicated step
app.post('/text-processor', (req, res) => {
  const { arguments: args } = req.body;

  res.json({
    success: true,
    outputs: {
      processed_text: args.input_text.toUpperCase()
    }
  });
});

// Multiple steps on same service
app.post('/validate-user', (req, res) => { /* ... */ });
app.post('/send-email', (req, res) => { /* ... */ });
app.post('/update-database', (req, res) => { /* ... */ });
```

### Go Service
```go
func textProcessorHandler(w http.ResponseWriter, r *http.Request) {
    var req StepRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", 400)
        return
    }

    inputText := req.Arguments["input_text"].(string)

    response := StepResponse{
        Success: true,
        Outputs: api.Args{
            "processed_text": strings.ToUpper(inputText),
            "processing_time_ms": 25,
        },
    }

    json.NewEncoder(w).Encode(response)
}
```

## Validation and Testing

### For Engine API Consumers
Use `engine-api.yaml` to:
1. **Generate Client SDKs**: Create type-safe engine clients in your preferred language
2. **Mock Engine**: Generate mock engine servers for testing
3. **Validation**: Ensure your integration matches the engine contract
4. **Documentation**: Generate interactive API docs for the engine

### For Step Implementors
Use `step-interface.yaml` to:
1. **Generate Server Stubs**: Create step endpoint templates in your preferred language
2. **Mock Testing**: Generate mock step servers for development
3. **Validation**: Ensure your step implementation matches the contract
4. **Documentation**: Generate interactive API docs for your steps

## Best Practices

### 1. Idempotency
Steps should be idempotent when possible - calling the same step with the same inputs should produce the same outputs.

### 2. Error vs Exception
- **Errors**: Business logic failures that might be recoverable
- **Exceptions**: Fatal conditions that should terminate the entire workflow

### 3. Timeout Handling
Respect optional argument timeouts and fail gracefully if external dependencies are unavailable.

### 4. Logging and Observability
Use the `meta` context for correlation IDs, user context, and distributed tracing.

### 5. Backward Compatibility
When updating steps, maintain backward compatibility with existing workflows or use versioning.

## Tools and Utilities

- **Swagger Editor**: Edit and validate the OpenAPI spec
- **OpenAPI Generator**: Generate client SDKs and server stubs
- **Postman**: Import the spec for interactive testing
- **Spuds CLI**: Tools for step registration and testing (coming soon)

## Contributing

### Changes to Engine API
When proposing changes to the engine API:
1. Update `engine-api.yaml` with your changes
2. Update server implementation in `internal/server/`
3. Add examples showing the new functionality
4. Update this README with any new endpoints or patterns
5. Ensure backward compatibility or provide migration guidance

### Changes to Step Interface
When proposing changes to the step interface:
1. Update `step-interface.yaml` with your changes
2. Provide examples showing the new functionality
3. Update this README with any new patterns or best practices
4. Ensure backward compatibility or provide migration guidance
5. Consider impact on existing step implementations
