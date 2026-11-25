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
                required: [customerId]
                properties:
                  customerId:
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
	assert.NotContains(t, summary, "recommended_steps")
}

func TestAnalyzeServiceLandscape(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "analyze_service_landscape", map[string]any{
		"services": []map[string]any{
			{"name": "customer-service", "spec_text": customerServiceOpenAPI},
			{"name": "order-service", "spec_text": orderServiceOpenAPI},
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

func TestGenerateStepImplScript(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "generate_step_impl", map[string]any{
		"language": "go",
		"step": map[string]any{
			"id":   "normalize-customer-age",
			"name": "Normalize Customer Age",
			"type": "script",
			"script": map[string]any{
				"language": "lua",
				"source":   "return {customer_age = customer_age}",
			},
			"attributes": map[string]any{
				"customer_age":        map[string]any{"role": "required"},
				"target_customer_age": map[string]any{"role": "output"},
			},
		},
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)
	assert.Equal(t, "script", payload["step_type"])

	code := payload["code"].(string)
	assert.Contains(t, code, "json.Unmarshal")
	assertFormattedGo(t, code)
	assert.Contains(t, code, "Register")
}

func TestGenerateStepImplHTTPConfiguredStepPreservesType(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "generate_step_impl", map[string]any{
		"language": "go",
		"step": map[string]any{
			"id":   "lookup-user",
			"name": "Lookup User",
			"type": "service",
			"http": map[string]any{
				"invoke": map[string]any{
					"endpoint": "http://users/{user_id}",
					"method":   "GET",
					"mode":     "async",
				},
			},
		},
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)
	assert.Equal(t, "service", payload["step_type"])
	assert.Equal(t, "async", payload["action_mode"])

	code := payload["code"].(string)
	registration := payload["registration"].(map[string]any)
	invoke := registration["http"].(map[string]any)["invoke"].(map[string]any)
	assert.Equal(t, "GET", invoke["method"])
	assert.Equal(t, "async", invoke["mode"])
	assert.Contains(t, code, "Register")
}

func TestAnalyzeServiceLandscapeWithRegisteredSteps(t *testing.T) {
	hc := &http.Client{
		Transport: roundTripperFunc(
			func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/engine/steps" || r.Method != http.MethodGet {
					return jsonResponse(
						http.StatusNotFound, []byte(`{"error":"not found"}`),
					), nil
				}
				return jsonResponse(http.StatusOK, []byte(`{
					"steps": [
						{
							"id": "lookup-customer-by-email",
							"name": "Lookup Customer By Email",
							"attributes": {
								"customer_email": {
									"role": "required"
								},
								"customer_id": {
									"role": "output"
								}
							}
						}
					],
					"count": 1
				}`)), nil
			},
		),
	}
	c := newClient(t, mcp.NewServer("http://example", hc))
	text := callToolText(t, c, "analyze_service_landscape", map[string]any{
		"services": []map[string]any{{"name": "order-service", "spec_text": orderServiceOpenAPI}},
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, payload["registered_steps"])
}

func TestServiceAnalysisRejects(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	for tool, args := range map[string]map[string]any{
		"analyze_service_spec":      {"spec_text": "openapi: ["},
		"analyze_service_landscape": {"services": []map[string]any{}},
		"generate_step_impl":        {"language": "go"},
	} {
		assert.True(t, callToolFails(t, c, tool, args))
	}
	assert.True(t, callToolFails(t, c, "analyze_service_landscape", map[string]any{
		"services":           []map[string]any{{"spec_text": "openapi: ["}},
		"include_registered": false,
	}))
	assert.True(t, callToolFails(t, c, "generate_step_impl", map[string]any{
		"language": "rust",
		"step":     map[string]any{"id": "s"},
	}))
}

func TestAnalyzeServiceLandscapeNamesUntitledService(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "analyze_service_landscape", map[string]any{
		"services": []map[string]any{{
			"spec_text": "openapi: 3.0.3\ninfo: {title: \" \", version: \"1\"}\npaths: {}\n",
		}},
		"include_registered": false,
	})

	var payload struct {
		Services []struct {
			Name string `json:"name"`
		} `json:"services"`
	}
	assert.NoError(t, json.Unmarshal([]byte(text), &payload))
	assert.Equal(t, "service", payload.Services[0].Name)
}

func TestGenerateStepImplPythonScript(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "generate_step_impl", map[string]any{
		"language": "python",
		"step": map[string]any{
			"id":     "tally",
			"name":   "tally",
			"type":   "script",
			"script": map[string]any{"language": "argyll", "source": "{}"},
			"attributes": map[string]any{
				"count": map[string]any{"role": "output"},
			},
		},
	})

	var payload map[string]any
	assert.NoError(t, json.Unmarshal([]byte(text), &payload))
	assert.Equal(t, "tally", payload["step_name"])
	assert.Equal(t, "script", payload["step_type"])
	registration := payload["registration"].(map[string]any)
	assert.Equal(t, "argyll", registration["script"].(map[string]any)["language"])
}

func TestGenerateStepImplRequiresScriptLanguage(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolErrorText(t, c, "generate_step_impl", map[string]any{
		"language": "python",
		"step": map[string]any{
			"id": "tally", "name": "tally", "type": "script",
			"script": map[string]any{"source": "elapsed"},
		},
	})
	assert.Contains(t, text, "script_language must be lua or argyll")
}
