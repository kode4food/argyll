package mcp_test

import (
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
	for _, tc := range []struct {
		tool string
		args map[string]any
	}{
		{tool: "start_flow", args: map[string]any{"goals": empty}},
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
		res, err := c.CallTool(tc.tool, tc.args)
		assert.NoError(t, err, tc.tool)
		m, _ := res.(map[string]any)
		assert.Equal(t, true, m["isError"], tc.tool)
	}
}

func TestToolDescriptionsExplainArgs(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", http.DefaultClient))
	tools, err := c.ListTools()
	assert.NoError(t, err)

	desc := map[string]string{}
	props := map[string]map[string]any{}
	for _, tool := range tools {
		desc[tool.Name] = tool.Description
		if p, ok := tool.InputSchema["properties"].(map[string]any); ok {
			props[tool.Name] = p
		}
	}
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
