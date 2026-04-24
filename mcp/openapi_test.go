package mcp_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp"
)

func TestInferOpenAPIStepsUsesExistingGraph(t *testing.T) {
	sample := fixtureText(t, "customer-orders-openapi.yaml")
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
							"type": "sync",
							"http": {
								"method": "GET",
								"endpoint": "http://example/customers/by-email"
							},
							"attributes": {
								"customer_email": {
									"role": "required",
									"type": "string"
								},
								"customer_id": {
									"role": "output",
									"type": "string"
								},
								"customer": {
									"role": "output",
									"type": "object"
								}
							}
						}
					},
					"count": 1
				}`)), nil
			},
		),
	}
	c := newClient(t, mcp.NewServer("http://example", hc))
	text := callToolText(t, c, "infer_openapi_steps", map[string]any{
		"spec_text": sample,
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)

	recommended, ok := payload["recommended_steps"].([]any)
	if !assert.True(t, ok) {
		return
	}
	assert.Len(t, recommended, 2)

	plans, ok := payload["plans"].([]any)
	if !assert.True(t, ok) || !assert.NotEmpty(t, plans) {
		return
	}
	found := false
	for _, item := range plans {
		plan, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if plan["goal_step_id"] != "create-order" {
			continue
		}
		steps, ok := plan["steps"].([]any)
		if !assert.True(t, ok) {
			return
		}
		assert.Len(t, steps, 2)
		first := steps[0].(map[string]any)
		second := steps[1].(map[string]any)
		assert.Equal(t, "lookup-customer-by-email", first["id"])
		assert.Equal(t, "existing", first["source"])
		assert.Equal(t, "create-order", second["id"])

		missing := plan["missing_inputs"].([]any)
		assert.Equal(t, []any{"customer_email", "items"}, missing)
		found = true
	}
	assert.True(t, found)
}

func TestInferOpenAPIProposeRegistrationsMode(t *testing.T) {
	sample := fixtureText(t, "customer-orders-openapi.yaml")
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "infer_openapi_steps", map[string]any{
		"mode":               "propose_registrations",
		"spec_text":          sample,
		"include_registered": false,
		"existing_steps": map[string]any{
			"steps": map[string]any{
				"create-order-lite": map[string]any{
					"id":   "create-order-lite",
					"name": "Create Order Lite",
					"attributes": map[string]any{
						"customer_id": map[string]any{
							"role": "required",
							"type": "string",
						},
						"items": map[string]any{
							"role": "required",
							"type": "array",
						},
						"order_id": map[string]any{
							"role": "output",
							"type": "string",
						},
					},
				},
			},
		},
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)
	assert.Equal(t, "propose_registrations", payload["mode"])

	props, ok := payload["proposed_registrations"].([]any)
	if !assert.True(t, ok) {
		return
	}
	assert.Len(t, props, 2)
	assert.NotNil(t, payload["llm_handoff_prompt"])
}

func TestInferOpenAPIStepsNestedSharedSchema(t *testing.T) {
	nested := fixtureText(t, "nested-customer-openapi.yaml")
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "infer_openapi_steps", map[string]any{
		"spec_text":          nested,
		"include_registered": false,
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)

	candidates, ok := payload["candidate_steps"].([]any)
	if !assert.True(t, ok) {
		return
	}
	if !assert.NotEmpty(t, candidates) {
		return
	}
	first := candidates[0].(map[string]any)
	step := first["step"].(map[string]any)
	attrs := step["attributes"].(map[string]any)

	customerID := attrs["customer_id"].(map[string]any)
	assert.Equal(t, "output", customerID["role"])
	mapping := customerID["mapping"].(map[string]any)
	script := mapping["script"].(map[string]any)
	assert.Equal(t, "$.data.id", script["script"])

	rationale := first["rationale"].([]any)
	assert.NotEmpty(t, rationale)
}

func TestInferOpenAPICoveragePartialOverlap(t *testing.T) {
	sample := fixtureText(t, "customer-orders-openapi.yaml")
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "infer_openapi_steps", map[string]any{
		"spec_text":          sample,
		"include_registered": false,
		"existing_steps": map[string]any{
			"steps": map[string]any{
				"list-orders-alt": map[string]any{
					"id":   "list-orders-alt",
					"name": "List Orders Alt",
					"attributes": map[string]any{
						"customer_id": map[string]any{
							"role": "required",
							"type": "string",
						},
						"order_summary": map[string]any{
							"role": "output",
							"type": "array",
						},
					},
				},
			},
		},
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)

	candidates := payload["candidate_steps"].([]any)
	found := false
	for _, raw := range candidates {
		item := raw.(map[string]any)
		if item["id"] != "list-customer-orders" {
			continue
		}
		assert.Equal(t, "recommended", item["coverage_status"])
		found = true
	}
	assert.True(t, found)
}
