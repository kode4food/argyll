package mcp_test

import (
	"bytes"
	"encoding/json"
	"go/format"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/localrivet/gomcp/client"
	"github.com/localrivet/gomcp/transport/embedded"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func TestPreviewPlanTool(t *testing.T) {
	hc := &http.Client{
		Transport: roundTripperFunc(
			func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/engine/plan" || r.Method != http.MethodPost {
					return jsonResponse(
						http.StatusNotFound, []byte(`{"error":"not found"}`),
					), nil
				}
				return jsonResponse(
					http.StatusOK,
					[]byte(`{"goals":["goal"],"required":["input"]}`),
				), nil
			},
		),
	}
	c := newClient(t, mcp.NewServer("http://example", hc))
	text := callToolText(t, c, "preview_plan", map[string]any{
		"goals": []string{"goal"},
	})
	assert.JSONEq(t, `{"goals":["goal"],"required":["input"]}`, text)
}

func TestNewServerTrimsTrailingSlash(t *testing.T) {
	hc := &http.Client{
		Transport: roundTripperFunc(
			func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "/engine/step", r.URL.Path)
				return jsonResponse(
					http.StatusOK,
					[]byte(`{"steps":[]}`),
				), nil
			},
		),
	}
	c := newClient(t, mcp.NewServer("http://example/", hc))
	_ = callToolText(t, c, "list_steps", map[string]any{})
}

func TestQueryFlowsTool(t *testing.T) {
	want := `{"flows":[{"id":"wf-1","status":"active"}],"count":1}`

	hc := &http.Client{
		Transport: roundTripperFunc(
			func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/engine/flow/query" ||
					r.Method != http.MethodPost {
					return jsonResponse(
						http.StatusNotFound,
						[]byte(`{"error":"not found"}`),
					), nil
				}

				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				var req map[string]any
				err = json.Unmarshal(b, &req)
				assert.NoError(t, err)
				assert.Equal(t, "wf-", req["id_prefix"])
				assert.Equal(t, float64(25), req["limit"])
				assert.Equal(t, "recent_desc", req["sort"])
				assert.Equal(t, []any{"active", "failed"}, req["statuses"])

				return jsonResponse(
					http.StatusOK,
					[]byte(want),
				), nil
			},
		),
	}
	c := newClient(t, mcp.NewServer("http://example", hc))
	text := callToolText(t, c, "query_flows", map[string]any{
		"id_prefix": "wf-",
		"statuses":  []string{"active", "failed"},
		"limit":     25,
		"sort":      "recent_desc",
	})
	assert.JSONEq(t, want, text)
}

func TestGetFlowStatusTool(t *testing.T) {
	hc := &http.Client{
		Transport: roundTripperFunc(
			func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/engine/flow/wf-123/status" ||
					r.Method != http.MethodGet {
					return jsonResponse(
						http.StatusNotFound, []byte(`{"error":"not found"}`),
					), nil
				}
				return jsonResponse(
					http.StatusOK,
					[]byte(`{"id":"wf-123","status":"completed"}`),
				), nil
			},
		),
	}
	c := newClient(t, mcp.NewServer("http://example", hc))
	text := callToolText(t, c, "get_flow_status", map[string]any{
		"id": "wf-123",
	})
	assert.JSONEq(t, `{"id":"wf-123","status":"completed"}`, text)
}

func TestOpenAPIHTTPMethod(t *testing.T) {
	c := newClient(t, mcp.NewServer("http://example", nil))
	text := callToolText(t, c, "openapi", map[string]any{})

	assert.Contains(t, text, `"method"`)
	assert.Contains(t, text, `"GET"`)
	assert.Contains(t, text, `"POST"`)
	assert.Contains(t, text, `"PUT"`)
	assert.Contains(t, text, `"DELETE"`)
	assert.Contains(t, text, "Defaults to `POST` when omitted")
}

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
				formatted, err := format.Source([]byte(code))
				assert.NoError(t, err)
				assert.Equal(t, string(formatted), code)
				return
			}
			assertPython(t, tt.name, code)
		})
	}
}

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newClient(t *testing.T, s *mcp.Server) client.Client {
	serverTr, clientTr := embedded.NewTransportPair()
	srv := s.MCPServer().AsEmbedded(serverTr)
	go func() {
		_ = srv.Run()
	}()

	c, err := client.NewClient("embedded://", client.WithEmbedded(clientTr))
	assert.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func callToolText(
	t *testing.T, c client.Client, name string, args map[string]any,
) string {
	res, err := c.CallTool(name, args)
	if !assert.NoError(t, err) {
		return ""
	}

	m, ok := res.(map[string]any)
	if !assert.True(t, ok) {
		return ""
	}

	content, ok := m["content"].([]any)
	if !assert.True(t, ok) {
		return ""
	}
	if !assert.Len(t, content, 1) {
		return ""
	}

	item, ok := content[0].(map[string]any)
	if !assert.True(t, ok) {
		return ""
	}
	assert.Equal(t, "text", item["type"])

	text, ok := item["text"].(string)
	if !assert.True(t, ok) {
		return ""
	}
	return text
}

func getResourceText(t *testing.T, c client.Client, uri string) string {
	res, err := c.GetResource(uri)
	if !assert.NoError(t, err) {
		return ""
	}
	if len(res.Content) > 0 {
		return strings.TrimSpace(res.Content[0].Text)
	}
	if len(res.Contents) > 0 {
		return strings.TrimSpace(res.Contents[0].Text)
	}
	return ""
}

func templateCode(
	t *testing.T, c client.Client, args map[string]any,
) string {
	text := callToolText(t, c, "sdk_step_template", args)
	var payload struct {
		Code string `json:"code"`
	}
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)
	return payload.Code
}

func assertWidth(t *testing.T, text string, limit int) {
	for i, line := range strings.Split(text, "\n") {
		expanded := strings.ReplaceAll(line, "\t", "    ")
		assert.LessOrEqual(t, len(expanded), limit, "line %d: %s", i+1, line)
	}
}

func assertPython(t *testing.T, name, code string) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available")
	}
	path := filepath.Join(t.TempDir(), name+".py")
	err = os.WriteFile(path, []byte(code), 0o600)
	assert.NoError(t, err)
	cmd := exec.Command(python, "-m", "py_compile", path)
	out, err := cmd.CombinedOutput()
	assert.NoError(t, err, string(out))
}

func jsonResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header: http.Header{
			"Content-Type": []string{
				"application/json",
			},
		},
		Body: io.NopCloser(bytes.NewReader(body)),
	}
}
