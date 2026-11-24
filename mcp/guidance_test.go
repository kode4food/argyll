package mcp_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp"
)

func TestSDKGuidance(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))

	text := getResourceText(t, c, "/sdk/steps")
	assert.Contains(t, text, "Supported configured HTTP methods")
	assert.Contains(t, text, "SDK-hosted servers handle POST step invocations")

	goText := getResourceText(t, c, "/sdk/go/steps")
	assert.Contains(t, goText, "WithMethod(\"GET\")")

	pyText := getResourceText(t, c, "/sdk/python/steps")
	assert.Contains(t, pyText, ".with_method(\"GET\")")

	openAPIText := getResourceText(t, c, "/openapi/ingestion")
	assert.Contains(t, openAPIText, "schema facts")
	assert.Contains(t, openAPIText, "required.match")
}

func TestStepPrompt(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))

	text := getPromptText(t, c, "implement_step", map[string]string{
		"language":     "Go",
		"step_name":    "Lookup User",
		"requirements": "Fetch a user by id.",
	})
	assert.Contains(t, text, "Lookup User")
	assert.Contains(t, text, "Fetch a user by id.")
	assert.Contains(t, text, "WithMethod/with_method")
}

func TestOpenAPIIngestionPrompt(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))

	text := getPromptText(t, c, "ingest_openapi_services", map[string]string{
		"payload": "{}",
	})
	assert.Contains(t, text, "schema facts")
	assert.Contains(t, text, "{}")
}

func TestStepTemplate(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "sdk_step_template", map[string]any{
		"language":  "go",
		"step_name": "Lookup User",
		"step_type": "service",
		"external":  true,
		"method":    "GET",
		"inputs":    []string{"user_id"},
		"outputs":   []string{"user"},
	})

	assert.Contains(t, text, "WithMethod(\\\"GET\\\")")
	assert.Contains(t, text, "Required(\\\"user_id\\\")")
	assert.Contains(t, text, "Register(\\n\\t\\tcontext.Background(),")
}

func TestStepTemplateRejectsExternalStepType(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolErrorText(t, c, "sdk_step_template", map[string]any{
		"language":  "go",
		"step_name": "Lookup User",
		"step_type": "external",
		"method":    "GET",
		"inputs":    []string{"user_id"},
		"outputs":   []string{"user"},
	})

	assert.Contains(t, text, "step_type must identify an SDK-implemented")
}

func TestStepTemplateRejectsExternalScript(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolErrorText(t, c, "sdk_step_template", map[string]any{
		"language":  "go",
		"step_name": "Calculate",
		"step_type": "script",
		"external":  true,
		"method":    "POST",
		"inputs":    []string{},
		"outputs":   []string{},
	})

	assert.Contains(t, text, "external is only valid for service")
}

func TestStepTemplateFormat(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	cases := []struct {
		name string
		args map[string]any
	}{
		{
			name: "go_sync",
			args: map[string]any{
				"language":  "go",
				"step_name": "Greeting",
				"step_type": "service",
				"method":    "POST",
				"inputs":    []string{"name"},
				"outputs":   []string{"greeting"},
			},
		},
		{
			name: "go_async",
			args: map[string]any{
				"language":    "go",
				"step_name":   "Async Task",
				"step_type":   "service",
				"action_mode": "async",
				"method":      "POST",
				"inputs":      []string{},
				"outputs":     []string{"status"},
			},
		},
		{
			name: "go_external_endpoint",
			args: map[string]any{
				"language":  "go",
				"step_name": "Lookup User",
				"step_type": "service",
				"external":  true,
				"method":    "GET",
				"inputs":    []string{"user_id"},
				"outputs":   []string{"user"},
			},
		},
		{
			name: "python_sync",
			args: map[string]any{
				"language":  "python",
				"step_name": "Greeting",
				"step_type": "service",
				"method":    "POST",
				"inputs":    []string{"name"},
				"outputs":   []string{"greeting"},
			},
		},
		{
			name: "python_external_endpoint",
			args: map[string]any{
				"language":  "python",
				"step_name": "Lookup User",
				"step_type": "service",
				"external":  true,
				"method":    "GET",
				"inputs":    []string{"user_id"},
				"outputs":   []string{"user"},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			code := templateCode(t, c, tt.args)
			assertWidth(t, code, 80)
			if tt.args["language"] == "go" {
				assertFormattedGo(t, code)
				return
			}
			assertPython(t, tt.name, code)
		})
	}
}

func TestStepTemplateRejects(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	for _, tc := range []struct {
		args map[string]any
		want string
	}{
		{
			args: map[string]any{"language": "rust", "step_name": "S"},
			want: "language must be go or python",
		},
		{
			args: map[string]any{"language": "go", "step_name": " "},
			want: "step_name is required",
		},
		{
			args: map[string]any{"language": "go", "step_name": "S", "step_type": "flow"},
			want: "flow steps do not have SDK handler implementations",
		},
		{
			args: map[string]any{"language": "go", "step_name": "S", "action_mode": "later"},
			want: "action_mode must be sync or async",
		},
		{
			args: map[string]any{"language": "go", "step_name": "S", "method": "PATCH"},
			want: "method must be GET, POST, PUT, or DELETE",
		},
		{
			args: map[string]any{
				"language":    "go",
				"step_name":   "S",
				"step_type":   "script",
				"action_mode": "sync",
			},
			want: "action_mode is only valid for service steps",
		},
		{
			args: map[string]any{"language": "go", "step_name": "S", "method": "GET"},
			want: "SDK-hosted service templates require method POST",
		},
	} {
		text := callToolErrorText(t, c, "sdk_step_template", tc.args)
		assert.Contains(t, text, tc.want)
	}
}
