package mcp_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp"
)

func TestDiffProposedStepsTool(t *testing.T) {
	hc := &http.Client{
		Transport: roundTripperFunc(
			func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/engine/step" || r.Method != http.MethodGet {
					return jsonResponse(
						http.StatusNotFound, []byte(`{"error":"not found"}`),
					), nil
				}
				return jsonResponse(http.StatusOK, []byte(`{
					"steps": [
						{
							"id": "same-step",
							"name": "Same Step",
							"type": "sync",
							"attributes": {
								"customer_id": {
									"role": "required",
									"type": "string"
								},
								"order_id": {
									"role": "output",
									"type": "string"
								}
							}
						},
						{
							"id": "update-step",
							"name": "Update Step",
							"type": "sync",
							"attributes": {
								"customer_id": {
									"role": "required",
									"type": "string"
								},
								"status": { "role": "output", "type": "string" }
							}
						}
					],
					"count": 2
				}`)), nil
			},
		),
	}
	c := newClient(t, mcp.NewServer("http://example", hc))
	text := callToolText(t, c, "diff_proposed_steps", map[string]any{
		"steps": []map[string]any{
			{
				"id":   "same-step",
				"name": "Same Step",
				"type": "sync",
				"attributes": map[string]any{
					"customer_id": map[string]any{
						"role": "required",
						"type": "string",
					},
					"order_id": map[string]any{
						"role": "output",
						"type": "string",
					},
				},
			},
			{
				"id":   "update-step",
				"name": "Update Step",
				"type": "sync",
				"attributes": map[string]any{
					"customer_id": map[string]any{
						"role": "required",
						"type": "string",
					},
					"order_id": map[string]any{
						"role": "output",
						"type": "string",
					},
				},
			},
			{
				"id":   "new-step",
				"name": "New Step",
				"type": "sync",
				"attributes": map[string]any{
					"customer_id": map[string]any{
						"role": "required",
						"type": "string",
					},
					"order_id": map[string]any{
						"role": "output",
						"type": "string",
					},
				},
			},
		},
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)

	summary := payload["summary"].(map[string]any)
	assert.Equal(t, float64(1), summary["create"])
	assert.Equal(t, float64(1), summary["update"])
	assert.Equal(t, float64(1), summary["skip"])

	items := payload["steps"].([]any)
	assert.Len(t, items, 3)
}

func TestApplyProposedStepsTool(t *testing.T) {
	var posts []map[string]any
	var puts []map[string]any

	hc := &http.Client{
		Transport: roundTripperFunc(
			func(r *http.Request) (*http.Response, error) {
				switch {
				case r.URL.Path == "/engine/step" && r.Method == http.MethodGet:
					return jsonResponse(http.StatusOK, []byte(`{
						"steps": [
							{
								"id": "same-step",
								"name": "Same Step",
								"type": "sync",
								"attributes": {
									"customer_id": {
										"role": "required",
										"type": "string"
									},
									"order_id": {
										"role": "output",
										"type": "string"
									}
								}
							},
							{
								"id": "update-step",
								"name": "Update Step",
								"type": "sync",
								"attributes": {
									"customer_id": {
										"role": "required",
										"type": "string"
									},
									"status": {
										"role": "output",
										"type": "string"
									}
								}
							}
						],
						"count": 2
					}`)), nil
				case r.URL.Path == "/engine/step" &&
					r.Method == http.MethodPost:
					var body map[string]any
					data, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					err = json.Unmarshal(data, &body)
					assert.NoError(t, err)
					posts = append(posts, body)
					return jsonResponse(
						http.StatusCreated,
						[]byte(`{"message":"created"}`),
					), nil
				case r.URL.Path == "/engine/step/update-step" &&
					r.Method == http.MethodPut:
					var body map[string]any
					data, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					err = json.Unmarshal(data, &body)
					assert.NoError(t, err)
					puts = append(puts, body)
					return jsonResponse(
						http.StatusOK,
						[]byte(`{"message":"updated"}`),
					), nil
				default:
					return jsonResponse(
						http.StatusNotFound, []byte(`{"error":"not found"}`),
					), nil
				}
			},
		),
	}

	c := newClient(t, mcp.NewServer("http://example", hc))
	text := callToolText(t, c, "apply_proposed_steps", map[string]any{
		"steps": []map[string]any{
			{
				"id":   "same-step",
				"name": "Same Step",
				"type": "sync",
				"attributes": map[string]any{
					"customer_id": map[string]any{
						"role": "required",
						"type": "string",
					},
					"order_id": map[string]any{
						"role": "output",
						"type": "string",
					},
				},
			},
			{
				"id":   "update-step",
				"name": "Update Step",
				"type": "sync",
				"attributes": map[string]any{
					"customer_id": map[string]any{
						"role": "required",
						"type": "string",
					},
					"order_id": map[string]any{
						"role": "output",
						"type": "string",
					},
				},
			},
			{
				"id":   "new-step",
				"name": "New Step",
				"type": "sync",
				"attributes": map[string]any{
					"customer_id": map[string]any{
						"role": "required",
						"type": "string",
					},
					"order_id": map[string]any{
						"role": "output",
						"type": "string",
					},
				},
			},
		},
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)
	assert.Len(t, posts, 1)
	assert.Len(t, puts, 1)

	summary := payload["summary"].(map[string]any)
	assert.Equal(t, float64(2), summary["applied"])
	assert.Equal(t, float64(1), summary["skipped"])
}

func TestApplyProposedStepsVerifiesSemanticReadback(t *testing.T) {
	gets := 0
	hc := &http.Client{
		Transport: roundTripperFunc(
			func(r *http.Request) (*http.Response, error) {
				switch {
				case r.URL.Path == "/engine/step" && r.Method == http.MethodGet:
					gets++
					if gets == 1 {
						return jsonResponse(
							http.StatusOK,
							[]byte(`{"steps":[],"count":0}`),
						), nil
					}
					return jsonResponse(http.StatusOK, []byte(`{
						"steps": [
							{
								"id": "send-sms",
								"name": "Send SMS",
								"type": "async",
								"attributes": {
									"body": {
										"role": "required",
										"type": "string"
									}
								}
							}
						],
						"count": 1
					}`)), nil
				case r.URL.Path == "/engine/step" &&
					r.Method == http.MethodPost:
					return jsonResponse(
						http.StatusCreated,
						[]byte(`{"message":"created"}`),
					), nil
				default:
					return jsonResponse(
						http.StatusNotFound, []byte(`{"error":"not found"}`),
					), nil
				}
			},
		),
	}

	c := newClient(t, mcp.NewServer("http://example", hc))
	text := callToolText(t, c, "apply_proposed_steps", map[string]any{
		"steps": []map[string]any{
			{
				"id":   "send-sms",
				"name": "Send SMS",
				"type": "async",
				"attributes": map[string]any{
					"body": map[string]any{
						"role": "required",
						"type": "string",
						"required": map[string]any{
							"mapping": map[string]any{
								"name": "message_text",
							},
						},
					},
				},
			},
		},
	})

	var payload map[string]any
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)

	summary := payload["summary"].(map[string]any)
	assert.Equal(t, "failed", summary["verification_status"])
	assert.Equal(t, float64(1), summary["verification_issues"])

	verification := payload["verification"].(map[string]any)
	issues := verification["issues"].([]any)
	issue := issues[0].(map[string]any)
	assert.Equal(t, "send-sms", issue["step_id"])
	assert.Equal(t, "attributes.body.required.mapping", issue["path"])
}
