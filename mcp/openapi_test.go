package mcp_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp"
)

func TestAnalyzeOpenAPIReturnsFacts(t *testing.T) {
	sample := fixtureText(t, "customer-orders-openapi.yaml")
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "analyze_openapi_contract", map[string]any{
		"spec_text":          sample,
		"include_registered": false,
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)
	assert.Equal(t, "contract", payload["mode"])
	assert.NotContains(t, payload, "proposed_registrations")
	assert.NotContains(t, payload, "candidate_steps")
	assert.NotContains(t, payload, "recommended_steps")

	ops := payload["operations"].([]any)
	assert.NotEmpty(t, ops)
	for _, rawOp := range ops {
		op := rawOp.(map[string]any)
		inputs := op["inputs"].([]any)
		for _, rawArg := range inputs {
			arg := rawArg.(map[string]any)
			assert.NotContains(t, arg, "mapping")
		}
	}

	handoff := payload["llm_handoff_prompt"].(string)
	assert.Contains(t, handoff, "analyze_openapi_contract")
	assert.NotContains(t, handoff, "infer_openapi_steps")
}

func TestAnalyzeOpenAPIObjectResponse(t *testing.T) {
	spec := `
openapi: 3.0.3
info:
  title: Customer Service
  version: 1.0.0
paths:
  /get-customer-info:
    post:
      operationId: getCustomerInfo
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [customer_id]
              properties:
                customer_id:
                  type: string
      responses:
        "200":
          description: customer found
          content:
            application/json:
              schema:
                type: object
                required: [customer_id, email_address, channel_preference]
                properties:
                  customer_id:
                    type: string
                  email_address:
                    type: string
                  channel_preference:
                    type: string
`
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "analyze_openapi_contract", map[string]any{
		"spec_text":          spec,
		"include_registered": false,
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)

	ops := payload["operations"].([]any)
	outputs := ops[0].(map[string]any)["outputs"].([]any)
	names := argNames(outputs)
	assert.Contains(t, names, "customer_id")
	assert.Contains(t, names, "email_address")
	assert.Contains(t, names, "channel_preference")
	assert.NotContains(t, names, "customer")
	assert.NotContains(t, names, "get_customer_info")
}

func TestAnalyzeOpenAPIHealth(t *testing.T) {
	spec := `
openapi: 3.0.3
info:
  title: Customer Service
  version: 1.0.0
servers:
  - url: http://example.com
paths:
  /health:
    get:
      operationId: health
      responses:
        "200":
          description: healthy
  /get-template:
    post:
      operationId: getTemplate
      responses:
        "200":
          description: template found
          content:
            application/json:
              schema:
                type: object
                properties:
                  template_id:
                    type: string
`
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "analyze_openapi_contract", map[string]any{
		"spec_text":          spec,
		"include_registered": false,
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)

	ops := payload["operations"].([]any)
	assert.Len(t, ops, 1)
	assert.Equal(t, "get-template", ops[0].(map[string]any)["id"])
}

func TestAnalyzeOpenAPIEndpointArgs(t *testing.T) {
	spec := `
openapi: 3.0.3
info:
  title: Message Service
  version: 1.0.0
servers:
  - url: http://example.com
paths:
  /customers/{customerId}/messages:
    get:
      operationId: listMessages
      parameters:
        - name: customerId
          in: path
          required: true
          schema:
            type: string
        - name: messageType
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          description: messages
          content:
            application/json:
              schema:
                type: object
                properties:
                  messages:
                    type: array
                    items:
                      type: object
`
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "analyze_openapi_contract", map[string]any{
		"spec_text":          spec,
		"include_registered": false,
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)

	ops := payload["operations"].([]any)
	op := ops[0].(map[string]any)
	assert.Equal(t,
		"http://example.com/customers/{customerId}/messages"+
			"?messageType={messageType}",
		op["endpoint"],
	)
	inputs := op["inputs"].([]any)
	assert.Contains(t, argNames(inputs), "customer_id")
	assert.Contains(t, argNames(inputs), "message_type")
}

func TestAnalyzeOpenAPIEnumFactsDoNotCreateMatches(t *testing.T) {
	spec := `
openapi: 3.0.3
info:
  title: Sender Service
  version: 1.0.0
paths:
  /send-sms:
    post:
      operationId: sendSMS
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [channelPreference, phoneNumber]
              properties:
                channelPreference:
                  type: string
                  enum: [sms]
                phoneNumber:
                  type: string
      responses:
        "202":
          description: accepted
`
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "analyze_openapi_contract", map[string]any{
		"spec_text":          spec,
		"include_registered": false,
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)

	ops := payload["operations"].([]any)
	inputs := ops[0].(map[string]any)["inputs"].([]any)
	found := false
	for _, raw := range inputs {
		arg := raw.(map[string]any)
		if arg["name"] != "channel_preference" {
			continue
		}
		assert.NotContains(t, arg, "match")
		assert.NotContains(t, arg, "mapping")
		schema := arg["schema"].(map[string]any)
		assert.Equal(t, []any{"sms"}, schema["enum"])
		found = true
	}
	assert.True(t, found)
}

func TestAnalyzeOpenAPIAsyncCallbackOutputs(t *testing.T) {
	spec := `
openapi: 3.0.3
info:
  title: Sender Service
  version: 1.0.0
paths:
  /send-sms:
    post:
      operationId: sendSMS
      parameters:
        - name: Argyll-Webhook-URL
          in: header
          required: true
          schema:
            type: string
      responses:
        "202":
          description: accepted
      callbacks:
        deliveryReceipt:
          '{$request.header.Argyll-Webhook-URL}':
            post:
              requestBody:
                required: true
                content:
                  application/json:
                    schema:
                      type: object
                      required: [receipt_id, channel]
                      properties:
                        receipt_id:
                          type: string
                        channel:
                          type: string
              responses:
                "200":
                  description: accepted
`
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "analyze_openapi_contract", map[string]any{
		"spec_text":          spec,
		"include_registered": false,
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)

	ops := payload["operations"].([]any)
	op := ops[0].(map[string]any)
	assert.NotContains(t, argNames(op["inputs"].([]any)), "argyll_webhook_url")
	outputs := argNames(op["outputs"].([]any))
	assert.Contains(t, outputs, "receipt_id")
	assert.Contains(t, outputs, "channel")
}

func argNames(args []any) []string {
	res := make([]string, 0, len(args))
	for _, raw := range args {
		arg := raw.(map[string]any)
		name, _ := arg["name"].(string)
		res = append(res, name)
	}
	return res
}
