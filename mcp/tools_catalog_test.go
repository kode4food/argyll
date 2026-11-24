package mcp_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp"
)

func TestRoutedTools(t *testing.T) {
	space := map[string]any{"id": "risk"}
	feed := map[string]any{"id": "total"}
	for _, tc := range []struct {
		tool   string
		args   map[string]any
		method string
		path   string
		body   string
	}{
		{
			tool: "start_flow",
			args: map[string]any{
				"id":    "f1",
				"goals": map[string]any{"steps": []string{"notify"}},
			},
			method: http.MethodPost,
			path:   "/engine/flows",
			body:   `{"id":"f1","goals":{"steps":["notify"]},"init":{}}`,
		},
		{
			tool: "start_flow",
			args: map[string]any{
				"id":       "f2",
				"goals":    map[string]any{"steps": []string{"a"}},
				"init":     map[string]any{"user_id": []string{"u1"}},
				"tags":     []string{"env:test"},
				"space_id": "payments",
			},
			method: http.MethodPost,
			path:   "/engine/flows",
			body: `{"id":"f2","goals":{"steps":["a"]},"init":{"user_id":["u1"]},` +
				`"tags":["env:test"],"space_id":"payments"}`,
		},
		{
			tool:   "query_flows",
			args:   map[string]any{"tags": []string{"env:test"}},
			method: http.MethodPost,
			path:   "/engine/flows/query",
			body:   `{"tags":["env:test"]}`,
		},
		{
			tool:   "get_flow_events",
			args:   map[string]any{"id": "f1"},
			method: http.MethodGet,
			path:   "/engine/flows/f1/events",
		},
		{
			tool:   "list_spaces",
			method: http.MethodGet,
			path:   "/engine/spaces",
		},
		{
			tool:   "get_space",
			args:   map[string]any{"id": "risk"},
			method: http.MethodGet,
			path:   "/engine/spaces/risk",
		},
		{
			tool:   "list_space_steps",
			args:   map[string]any{"id": "risk"},
			method: http.MethodGet,
			path:   "/engine/spaces/risk/steps",
		},
		{
			tool:   "preview_space",
			args:   map[string]any{"space": space},
			method: http.MethodPost,
			path:   "/engine/spaces/preview",
			body:   `{"id":"risk"}`,
		},
		{
			tool:   "register_space",
			args:   map[string]any{"space": space},
			method: http.MethodPost,
			path:   "/engine/spaces",
			body:   `{"id":"risk"}`,
		},
		{
			tool:   "update_space",
			args:   map[string]any{"id": "risk", "space": space},
			method: http.MethodPut,
			path:   "/engine/spaces/risk",
			body:   `{"id":"risk"}`,
		},
		{
			tool:   "unregister_space",
			args:   map[string]any{"id": "risk"},
			method: http.MethodDelete,
			path:   "/engine/spaces/risk",
		},
		{
			tool:   "list_feeds",
			method: http.MethodGet,
			path:   "/engine/feeds",
		},
		{
			tool:   "get_feed",
			args:   map[string]any{"id": "total"},
			method: http.MethodGet,
			path:   "/engine/feeds/total",
		},
		{
			tool:   "register_feed",
			args:   map[string]any{"feed": feed},
			method: http.MethodPost,
			path:   "/engine/feeds",
			body:   `{"id":"total"}`,
		},
		{
			tool:   "update_feed",
			args:   map[string]any{"id": "total", "feed": feed},
			method: http.MethodPut,
			path:   "/engine/feeds/total",
			body:   `{"id":"total"}`,
		},
		{
			tool:   "unregister_feed",
			args:   map[string]any{"id": "total"},
			method: http.MethodDelete,
			path:   "/engine/feeds/total",
		},
	} {
		t.Run(tc.tool+" "+tc.path, func(t *testing.T) {
			hc := &http.Client{
				Transport: roundTripperFunc(
					func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, tc.method, r.Method)
						assert.Equal(t, tc.path, r.URL.Path)
						if tc.body != "" {
							body, err := io.ReadAll(r.Body)
							assert.NoError(t, err)
							assert.JSONEq(t, tc.body, string(body))
						}
						return jsonResponse(http.StatusOK, []byte(`{"ok":true}`)), nil
					},
				),
			}
			c := newClient(t, mcp.NewServer("http://example", hc))
			args := tc.args
			if args == nil {
				args = map[string]any{}
			}
			assert.JSONEq(t, `{"ok":true}`, callToolText(t, c, tc.tool, args))
		})
	}
}

func TestToolsRejectMissingArgs(t *testing.T) {
	hc := &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			assert.Fail(t, "engine called")
			return jsonResponse(http.StatusOK, []byte(`{}`)), nil
		}),
	}
	c := newClient(t, mcp.NewServer("http://example", hc))
	empty := map[string]any{}
	noSteps := map[string]any{"steps": []string{}}
	for _, tc := range []struct {
		tool string
		args map[string]any
	}{
		{tool: "start_flow", args: map[string]any{"goals": empty}},
		{tool: "start_flow", args: map[string]any{"goals": noSteps}},
		{tool: "preview_plan", args: map[string]any{"goals": noSteps}},
		{tool: "register_step", args: map[string]any{"step": empty}},
		{tool: "update_step", args: map[string]any{"step": map[string]any{"id": "x"}}},
		{tool: "analyze_openapi_contract", args: map[string]any{
			"spec_text":          "openapi: [",
			"include_registered": false,
		}},
		{tool: "get_flow_events", args: map[string]any{"id": ""}},
		{tool: "get_space", args: map[string]any{"id": ""}},
		{tool: "list_space_steps", args: map[string]any{"id": ""}},
		{tool: "preview_space", args: map[string]any{"space": empty}},
		{tool: "register_space", args: map[string]any{"space": empty}},
		{tool: "update_space", args: map[string]any{"space": map[string]any{"id": "x"}}},
		{tool: "update_space", args: map[string]any{"id": "x", "space": empty}},
		{tool: "unregister_space", args: map[string]any{"id": ""}},
		{tool: "get_feed", args: map[string]any{"id": ""}},
		{tool: "register_feed", args: map[string]any{"feed": empty}},
		{tool: "update_feed", args: map[string]any{"feed": map[string]any{"id": "x"}}},
		{tool: "update_feed", args: map[string]any{"id": "x", "feed": empty}},
		{tool: "unregister_feed", args: map[string]any{"id": ""}},
	} {
		assert.True(t, callToolFails(t, c, tc.tool, tc.args), tc.tool)
	}
}

func TestToolsReportEngineFailures(t *testing.T) {
	hc := &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusInternalServerError, []byte(`{"error":"down"}`)), nil
		}),
	}
	c := newClient(t, mcp.NewServer("http://example", hc))
	step := map[string]any{
		"id":   "s",
		"name": "S",
		"type": "sync",
		"http": map[string]any{"endpoint": "http://svc/s"},
	}
	goals := map[string]any{"steps": []string{"s"}}
	space := map[string]any{
		"id":       "risk",
		"name":     "Risk",
		"selector": map[string]any{"language": "lua", "source": "return true"},
	}
	feed := map[string]any{"id": "total", "name": "Total"}
	for name, args := range map[string]map[string]any{
		"engine_state":              {},
		"analyze_service_landscape": {},
		"list_steps":                {},
		"get_step":                  {"id": "s"},
		"unregister_step":           {"id": "s"},
		"register_step":             {"step": step},
		"update_step":               {"id": "s", "step": step},
		"diff_proposed_steps":       {"steps": []any{step}},
		"apply_proposed_steps":      {"steps": []any{step}},
		"list_flows":                {},
		"get_flow":                  {"id": "f"},
		"get_flow_status":           {"id": "f"},
		"get_flow_events":           {"id": "f"},
		"query_flows":               {},
		"start_flow":                {"goals": goals},
		"preview_plan":              {"goals": goals},
		"list_step_health":          {},
		"get_step_health":           {"id": "s"},
		"list_spaces":               {},
		"get_space":                 {"id": "risk"},
		"list_space_steps":          {"id": "risk"},
		"unregister_space":          {"id": "risk"},
		"preview_space":             {"space": space},
		"register_space":            {"space": space},
		"update_space":              {"id": "risk", "space": space},
		"list_feeds":                {},
		"get_feed":                  {"id": "total"},
		"unregister_feed":           {"id": "total"},
		"register_feed":             {"feed": feed},
		"update_feed":               {"id": "total", "feed": feed},
	} {
		t.Run(name, func(t *testing.T) {
			assert.True(t, callToolFails(t, c, name, args))
		})
	}
}

func TestToolDescriptionsExplainArgs(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", http.DefaultClient))
	tools, err := c.ListTools(context.Background(), nil)
	assert.NoError(t, err)

	desc := map[string]string{}
	props := map[string]map[string]any{}
	for _, tool := range tools.Tools {
		desc[tool.Name] = tool.Description
		schema, _ := tool.InputSchema.(map[string]any)
		if p, ok := schema["properties"].(map[string]any); ok {
			props[tool.Name] = p
		}
	}
	goals, _ := props["start_flow"]["goals"].(map[string]any)
	goalProps, _ := goals["properties"].(map[string]any)
	assert.Contains(t, goalProps, "steps")
	assert.Contains(t, goalProps, "else")
	for _, tool := range []string{"start_flow", "preview_plan"} {
		assert.Contains(t, desc[tool], `{"steps"`)
		assert.Contains(t, desc[tool], "without outputs")
		assert.Contains(t, desc[tool], `{"user_id": ["user-123"]}`)
	}
	assert.Contains(t, desc["register_step"], "role")
	for _, name := range []string{"id", "init", "tags", "space_id"} {
		assert.Contains(t, props["start_flow"], name)
	}
	assert.Contains(t, props["query_flows"], "tags")
	assert.NotContains(t, props["query_flows"], "labels")
}
