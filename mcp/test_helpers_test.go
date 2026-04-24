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

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newClient(t *testing.T, s *mcp.Server) client.Client {
	t.Helper()

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
	t.Helper()

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
	t.Helper()

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

func templateCode(t *testing.T, c client.Client, args map[string]any) string {
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
			"Content-Type": []string{
				"application/json",
			},
		},
		Body: io.NopCloser(bytes.NewReader(body)),
	}
}
