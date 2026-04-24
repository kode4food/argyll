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
	assert.Contains(t, text, "SDK-hosted server currently handles POST")

	goText := getResourceText(t, c, "/sdk/go/steps")
	assert.Contains(t, goText, "WithMethod(\"GET\")")

	pyText := getResourceText(t, c, "/sdk/python/steps")
	assert.Contains(t, pyText, ".with_method(\"GET\")")
}

func TestStepPrompt(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))

	prompt, err := c.GetPrompt("implement_step", map[string]any{
		"language":     "Go",
		"step_name":    "Lookup User",
		"requirements": "Fetch a user by id.",
	})
	assert.NoError(t, err)
	if !assert.Len(t, prompt.Messages, 1) {
		return
	}
	text := prompt.Messages[0].Content.Text
	assert.Contains(t, text, "Lookup User")
	assert.Contains(t, text, "Fetch a user by id.")
	assert.Contains(t, text, "WithMethod/with_method")
}

func TestStepTemplate(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "sdk_step_template", map[string]any{
		"language":  "go",
		"step_name": "Lookup User",
		"step_type": "external",
		"method":    "GET",
		"inputs":    []string{"user_id"},
		"outputs":   []string{"user"},
	})

	assert.Contains(t, text, "WithMethod(\\\"GET\\\")")
	assert.Contains(t, text, "Required(\\\"user_id\\\", api.TypeString)")
	assert.Contains(t, text, "Register(\\n\\t\\tcontext.Background(),")
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
				"step_type": "sync",
				"method":    "POST",
				"inputs":    []string{"name"},
				"outputs":   []string{"greeting"},
			},
		},
		{
			name: "go_async",
			args: map[string]any{
				"language":  "go",
				"step_name": "Async Task",
				"step_type": "async",
				"method":    "POST",
				"inputs":    []string{},
				"outputs":   []string{"status"},
			},
		},
		{
			name: "go_external_get",
			args: map[string]any{
				"language":  "go",
				"step_name": "Lookup User",
				"step_type": "external",
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
				"step_type": "sync",
				"method":    "POST",
				"inputs":    []string{"name"},
				"outputs":   []string{"greeting"},
			},
		},
		{
			name: "python_external_get",
			args: map[string]any{
				"language":  "python",
				"step_name": "Lookup User",
				"step_type": "external",
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
