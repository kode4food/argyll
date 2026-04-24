package mcp_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp"
)

const customerServiceOpenAPI = `
openapi: 3.0.3
info:
  title: Customer Service
  version: 1.0.0
paths:
  /customers/by-email:
    get:
      operationId: lookupCustomerByEmail
      summary: Lookup Customer By Email
      parameters:
        - in: query
          name: email
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [id]
                properties:
                  id:
                    type: string
`

const orderServiceOpenAPI = `
openapi: 3.0.3
info:
  title: Order Service
  version: 1.0.0
paths:
  /orders:
    post:
      operationId: createOrder
      summary: Create Order
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [customerId, items]
              properties:
                customerId:
                  type: string
                items:
                  type: array
                  items:
                    type: object
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                type: object
                required: [id]
                properties:
                  id:
                    type: string
`

func TestAnalyzeServiceSpec(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "analyze_service_spec", map[string]any{
		"name":      "customer-service",
		"spec_text": customerServiceOpenAPI,
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)
	assert.Equal(t, "customer-service", payload["service_name"])

	summary := payload["summary"].(map[string]any)
	assert.Equal(t, float64(1), summary["operations"])
	assert.Equal(t, float64(1), summary["recommended_steps"])
}

func TestAnalyzeServiceLandscape(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "analyze_service_landscape", map[string]any{
		"services": []map[string]any{
			{
				"name":      "customer-service",
				"spec_text": customerServiceOpenAPI,
			},
			{
				"name":      "order-service",
				"spec_text": orderServiceOpenAPI,
			},
		},
		"include_registered": false,
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)

	relationships := payload["relationships"].([]any)
	assert.NotEmpty(t, relationships)

	missing := payload["missing_attributes"].([]any)
	assert.NotEmpty(t, missing)

}

func TestProposeBridgeStepsScript(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "propose_bridge_steps", map[string]any{
		"landscape": map[string]any{
			"bridge_opportunities": []map[string]any{
				{
					"kind":             "script_bridge",
					"source_service":   "profile-service",
					"source_step_id":   "load-profile",
					"source_attribute": "customer_age",
					"source_type":      "string",
					"target_service":   "customer-service",
					"target_step_id":   "score-customer",
					"target_attribute": "customer_age",
					"target_type":      "integer",
					"confidence":       "high",
					"rationale":        "type mismatch needs Lua",
				},
			},
		},
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)

	props := payload["proposed_bridge_steps"].([]any)
	assert.Len(t, props, 1)

	prop := props[0].(map[string]any)
	assert.Equal(t, "script_bridge", prop["kind"])

	step := prop["step"].(map[string]any)
	labels := step["labels"].(map[string]any)
	assert.Equal(t, "script_bridge", labels["argyll.bridge_kind"])

	script := step["script"].(map[string]any)
	assert.Equal(t, "lua", script["language"])
	assert.Contains(t, script["script"], "return {")
}

func TestGenerateStepImplScript(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "generate_step_impl", map[string]any{
		"language": "go",
		"step": map[string]any{
			"id":   "bridge-customer-age",
			"name": "Bridge Customer Age",
			"type": "script",
			"labels": map[string]any{
				"argyll.bridge_kind": "script_bridge",
			},
			"script": map[string]any{
				"language": "lua",
				"script":   "return {customer_age = customer_age}",
			},
			"attributes": map[string]any{
				"customer_age": map[string]any{
					"role": "required",
					"type": "string",
				},
				"target_customer_age": map[string]any{
					"role": "output",
					"type": "integer",
				},
			},
		},
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)
	assert.Equal(t, "script", payload["step_type"])

	code := payload["code"].(string)
	assert.Contains(t, code, "WithScriptLanguage")
	assert.Contains(t, code, "Register")
}

func TestAnalyzeServiceLandscapeWithRegisteredSteps(t *testing.T) {
	hc := &http.Client{
		Transport: roundTripperFunc(
			func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/engine/step" || r.Method != http.MethodGet {
					return jsonResponse(
						http.StatusNotFound, []byte(`{"error":"not found"}`),
					), nil
				}
				return jsonResponse(http.StatusOK, []byte(`{
					"steps": {
						"lookup-customer-by-email": {
							"id": "lookup-customer-by-email",
							"name": "Lookup Customer By Email",
							"attributes": {
								"customer_email": {
									"role": "required",
									"type": "string"
								},
								"customer_id": {
									"role": "output",
									"type": "string"
								}
							}
						}
					}
				}`)), nil
			},
		),
	}
	c := newClient(t, mcp.NewServer("http://example", hc))
	text := callToolText(t, c, "analyze_service_landscape", map[string]any{
		"services": []map[string]any{
			{
				"name":      "order-service",
				"spec_text": orderServiceOpenAPI,
			},
		},
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, payload["registered_steps"])
}
