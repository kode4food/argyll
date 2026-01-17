package mcp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/localrivet/gomcp/client"
	"github.com/localrivet/gomcp/transport/embedded"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func TestPreviewPlanTool(t *testing.T) {
	httpClient := &http.Client{
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
	s := mcp.NewServer("http://example", httpClient)

	serverTransport, clientTransport := embedded.NewTransportPair()
	srv := s.MCPServer().AsEmbedded(serverTransport)
	go func() {
		_ = srv.Run()
	}()

	c, err := client.NewClient(
		"embedded://", client.WithEmbedded(clientTransport),
	)
	assert.NoError(t, err)
	defer func() { _ = c.Close() }()

	result, err := c.CallTool("preview_plan", map[string]any{
		"goals": []string{"goal"},
	})
	assert.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)

	content, ok := resultMap["content"].([]any)
	assert.True(t, ok)
	assert.Len(t, content, 1)

	item, ok := content[0].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "text", item["type"])

	text, ok := item["text"].(string)
	assert.True(t, ok)

	var payload map[string]any
	err = json.Unmarshal([]byte(text), &payload)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []any{"goal"}, payload["goals"])
	assert.Equal(t, []any{"input"}, payload["required"])
}

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
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
