package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp"
)

func TestNormalizedStepDiff(t *testing.T) {
	for _, tool := range []string{"diff_proposed_steps", "apply_proposed_steps"} {
		t.Run(tool, func(t *testing.T) {
			writes := 0
			hc := &http.Client{Transport: roundTripperFunc(func(
				r *http.Request,
			) (*http.Response, error) {
				if r.Method != http.MethodGet {
					writes++
				}
				return jsonResponse(http.StatusOK, []byte(`{
					"steps":[{"id":"s","tags":["a","b"]}]
				}`)), nil
			})}
			c := newClient(t, mcp.NewServer("http://example", hc))
			text := callToolText(t, c, tool, map[string]any{
				"steps": []map[string]any{{"id": "s", "tags": []string{"b", "a", "b"}}},
			})
			var payload struct {
				Summary struct {
					Skip int `json:"skip"`
				} `json:"summary"`
			}
			assert.NoError(t, json.Unmarshal([]byte(text), &payload))
			assert.Equal(t, 1, payload.Summary.Skip)
			assert.Zero(t, writes)
		})
	}
}

func TestApplyPartialFailure(t *testing.T) {
	writes := 0
	hc := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, []byte(`{"steps":[]}`)), nil
		}
		writes++
		if writes == 2 {
			return jsonResponse(http.StatusConflict, []byte(`{"error":"conflict"}`)), nil
		}
		return jsonResponse(http.StatusCreated, []byte(`{}`)), nil
	})}
	c := newClient(t, mcp.NewServer("http://example", hc))
	text := callToolErrorText(t, c, "apply_proposed_steps", map[string]any{
		"steps": []map[string]any{{"id": "a"}, {"id": "b"}, {"id": "c"}},
	})
	var payload struct {
		Applied []map[string]any `json:"applied_steps"`
		Failed  map[string]any   `json:"failed_step"`
		Pending []map[string]any `json:"pending_steps"`
	}
	assert.NoError(t, json.Unmarshal([]byte(text), &payload))
	assert.Equal(t, 2, writes)
	if assert.Len(t, payload.Applied, 1) {
		assert.Equal(t, "a", payload.Applied[0]["id"])
	}
	assert.Equal(t, "b", payload.Failed["id"])
	if assert.Len(t, payload.Pending, 1) {
		assert.Equal(t, "c", payload.Pending[0]["id"])
	}
}

func TestProposedIDsRejectBeforeRequests(t *testing.T) {
	calls := 0
	hc := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return jsonResponse(http.StatusOK, []byte(`{"steps":[]}`)), nil
	})}
	c := newClient(t, mcp.NewServer("http://example", hc))
	for _, tool := range []string{"diff_proposed_steps", "apply_proposed_steps"} {
		for _, steps := range [][]map[string]any{
			{{"id": "a"}, {}},
			{{"id": "a"}, {"id": "a"}},
			{{"id": " a "}},
		} {
			assert.True(t, callToolFails(t, c, tool, map[string]any{"steps": steps}))
		}
	}
	assert.Zero(t, calls)
}

func TestDiffExplicitWorkConfig(t *testing.T) {
	for _, work := range []string{`{"max_retries":7}`, `{"max_retries":0}`} {
		t.Run(work, func(t *testing.T) {
			hc := &http.Client{Transport: roundTripperFunc(func(
				*http.Request,
			) (*http.Response, error) {
				return jsonResponse(http.StatusOK, []byte(`{
					"steps":[{"id":"s","work_config":`+work+`}]
				}`)), nil
			})}
			c := newClient(t, mcp.NewServer("http://example", hc))
			args := map[string]any{"steps": []map[string]any{{"id": "s"}}}
			text := callToolText(t, c, "diff_proposed_steps", args)
			var payload struct {
				Steps []struct {
					Action string   `json:"action"`
					Fields []string `json:"fields"`
				} `json:"steps"`
			}
			assert.NoError(t, json.Unmarshal([]byte(text), &payload))
			if assert.Len(t, payload.Steps, 1) {
				assert.Equal(t, "update", payload.Steps[0].Action)
				assert.Equal(t, []string{"work_config"}, payload.Steps[0].Fields)
			}
		})
	}
}

func TestApplyReadbackFailure(t *testing.T) {
	gets := 0
	hc := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			return jsonResponse(http.StatusCreated, []byte(`{}`)), nil
		}
		gets++
		if gets == 1 {
			return jsonResponse(http.StatusOK, []byte(`{"steps":[]}`)), nil
		}
		return jsonResponse(http.StatusServiceUnavailable, []byte(`{}`)), nil
	})}
	c := newClient(t, mcp.NewServer("http://example", hc))
	text := callToolErrorText(t, c, "apply_proposed_steps", map[string]any{
		"steps": []map[string]any{{
			"id": "s",
			"attributes": map[string]any{
				"out": map[string]any{
					"role":   "output",
					"output": map[string]any{"mapping": map[string]any{"name": "result"}},
				},
			},
		}},
	})
	var payload struct {
		Phase              string           `json:"phase"`
		Applied            []map[string]any `json:"applied_steps"`
		VerificationStatus string           `json:"verification_status"`
	}
	assert.NoError(t, json.Unmarshal([]byte(text), &payload))
	assert.Equal(t, "verification", payload.Phase)
	assert.Equal(t, "unknown", payload.VerificationStatus)
	if assert.Len(t, payload.Applied, 1) {
		assert.Equal(t, "s", payload.Applied[0]["id"])
	}
}

func TestRegistrationPreserved(t *testing.T) {
	st := map[string]any{}
	assert.NoError(t, json.Unmarshal([]byte(`{
		"id":"original-id","name":"Different Name","type":"service",
		"tags":["important"],"handling":"compensated",
		"http":{"invoke":{"endpoint":"https://service/order","mode":"async","method":"PUT"},
		"compensate":{"endpoint":"https://service/undo","method":"DELETE"},
		"health":"https://service/health"},
		"attributes":{"input":{"role":"optional","optional":{"collect":"all","for_each":true,
		"default":"[]","match":{"language":"argyll","source":"true"},
		"mapping":{"name":"body","selector":{"language":"lua","source":"return value"}}}}},
		"predicate":{"language":"argyll","source":"true"},
		"work_config":{"max_retries":3,"parallelism":2}
	}`), &st))
	raw, err := json.Marshal(st)
	assert.NoError(t, err)
	c := newClient(t, mcp.NewServer("http://example", nil))
	for _, lang := range []string{"go", "python"} {
		text := callToolText(t, c, "generate_step_impl", map[string]any{
			"step":     st,
			"language": lang,
		})
		var payload struct {
			Code         string         `json:"code"`
			Registration map[string]any `json:"registration"`
			Inputs       []string       `json:"inputs"`
		}
		assert.NoError(t, json.Unmarshal([]byte(text), &payload))
		assert.Equal(t, st, payload.Registration)
		assert.Equal(t, []string{"input"}, payload.Inputs)
		var decoded strings.Builder
		reading := false
		for line := range strings.SplitSeq(payload.Code, "\n") {
			if strings.Contains(line, "registration :=") ||
				strings.Contains(line, "registration = json.loads(") {
				reading = true
				continue
			}
			if !reading {
				continue
			}
			if !strings.HasPrefix(strings.TrimSpace(line), `"`) {
				break
			}
			literal := strings.TrimSuffix(strings.TrimSpace(line), " +")
			chunk, err := strconv.Unquote(literal)
			assert.NoError(t, err)
			decoded.WriteString(chunk)
		}
		assert.JSONEq(t, string(raw), decoded.String())
		assert.NotContains(t, payload.Code, "localhost:8081")
		assertWidth(t, payload.Code, 100)
		if lang == "go" {
			assertFormattedGo(t, payload.Code)
		} else {
			assertPython(t, "registration", payload.Code)
		}
	}
}

func TestEngineResourcesAndCapabilities(t *testing.T) {
	hc := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/engine/agents/capabilities", r.URL.Path)
		return jsonResponse(http.StatusOK, []byte(`{"providers":[]}`)), nil
	})}
	c := newClient(t, mcp.NewServer("http://example", hc))
	assert.JSONEq(t, `{"providers":[]}`, callToolText(t, c, "agent_capabilities", nil))
	res, err := c.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: "/engine/api"})
	assert.NoError(t, err)
	if assert.Len(t, res.Contents, 1) {
		assert.Equal(t, "application/yaml", res.Contents[0].MIMEType)
		assert.Contains(t, res.Contents[0].Text, "StepRegistration:")
	}
	text := getResourceText(t, c, "/engine/semantics")
	assert.Contains(t, text, "deactivated_at")
	assert.Contains(t, text, "zero values")
}
