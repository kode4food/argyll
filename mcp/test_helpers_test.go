package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"go/format"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/mcp"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newClient(t *testing.T, s *mcp.Server) *sdk.ClientSession {
	t.Helper()

	ctx := context.Background()
	serverTr, clientTr := sdk.NewInMemoryTransports()
	srv, err := s.MCPServer()
	if !assert.NoError(t, err) {
		return nil
	}
	ss, err := srv.Connect(ctx, serverTr, nil)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	t.Cleanup(func() { _ = ss.Close() })

	client := sdk.NewClient(&sdk.Implementation{
		Name:    "test",
		Version: "1",
	}, nil)
	cs, err := client.Connect(ctx, clientTr, nil)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func callTool(
	t *testing.T, c *sdk.ClientSession, name string, args map[string]any,
) *sdk.CallToolResult {
	t.Helper()

	res, err := c.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return res
}

func callToolText(t *testing.T, c *sdk.ClientSession, name string, args map[string]any) string {
	t.Helper()

	res := callTool(t, c, name, args)
	if !assert.False(t, res.IsError) {
		return ""
	}
	if !assert.Len(t, res.Content, 1) {
		return ""
	}
	text, ok := res.Content[0].(*sdk.TextContent)
	if !assert.True(t, ok) {
		return ""
	}
	return text.Text
}

func callToolFails(t *testing.T, c *sdk.ClientSession, name string, args map[string]any) bool {
	t.Helper()

	return callTool(t, c, name, args).IsError
}

func callToolErrorText(
	t *testing.T, c *sdk.ClientSession, name string, args map[string]any,
) string {
	t.Helper()

	res := callTool(t, c, name, args)
	if !assert.True(t, res.IsError) || !assert.NotEmpty(t, res.Content) {
		return ""
	}
	text, _ := res.Content[0].(*sdk.TextContent)
	if text == nil {
		return ""
	}
	return text.Text
}

func getResourceText(t *testing.T, c *sdk.ClientSession, uri string) string {
	t.Helper()

	res, err := c.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: uri})
	if !assert.NoError(t, err) || !assert.NotEmpty(t, res.Contents) {
		return ""
	}
	return strings.TrimSpace(res.Contents[0].Text)
}

func getPromptText(
	t *testing.T, c *sdk.ClientSession, name string, args map[string]string,
) string {
	t.Helper()

	res, err := c.GetPrompt(context.Background(), &sdk.GetPromptParams{
		Name:      name,
		Arguments: args,
	})
	if !assert.NoError(t, err) || !assert.Len(t, res.Messages, 1) {
		return ""
	}
	text, ok := res.Messages[0].Content.(*sdk.TextContent)
	if !assert.True(t, ok) {
		return ""
	}
	return text.Text
}

func templateCode(t *testing.T, c *sdk.ClientSession, args map[string]any) string {
	t.Helper()

	text := callToolText(t, c, "sdk_step_template", args)
	var payload struct {
		Code string `json:"code"`
	}
	err := json.Unmarshal([]byte(text), &payload)
	assert.NoError(t, err)
	return payload.Code
}

func fixtureText(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if !assert.NoError(t, err) {
		return ""
	}
	return string(data)
}

func assertWidth(t *testing.T, text string, limit int) {
	t.Helper()

	for i, line := range strings.Split(text, "\n") {
		expanded := strings.ReplaceAll(line, "\t", "    ")
		assert.LessOrEqual(t, len(expanded), limit, "line %d: %s", i+1, line)
	}
}

func assertPython(t *testing.T, name, code string) {
	t.Helper()

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

func assertFormattedGo(t *testing.T, code string) {
	t.Helper()

	formatted, err := format.Source([]byte(code))
	assert.NoError(t, err)
	assert.Equal(t, string(formatted), code)
}

func jsonResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header: http.Header{
			"Content-Type": []string{api.JSONContentType},
		},
		Body: io.NopCloser(bytes.NewReader(body)),
	}
}
