package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreviewPlanTool(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/engine/plan" || r.Method != http.MethodPost {
				return jsonResponse(
					http.StatusNotFound, []byte(`{"error":"not found"}`),
				), nil
			}
			return jsonResponse(
				http.StatusOK,
				[]byte(`{"goals":["goal"],"required":["input"]}`),
			), nil
		}),
	}
	server := NewServer("http://example", client)

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		server.ServeContext(ctx, inR, outW)
		close(done)
	}()

	enc := json.NewEncoder(inW)
	err := enc.Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      "1",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "preview_plan",
			"arguments": map[string]any{
				"goals": []string{"goal"},
			},
		},
	})
	assert.NoError(t, err)
	assert.NoError(t, inW.Close())

	dec := json.NewDecoder(outR)
	var resp map[string]any
	err = dec.Decode(&resp)
	assert.NoError(t, err)
	assert.Nil(t, resp["error"])

	result, ok := resp["result"].(map[string]any)
	assert.True(t, ok)

	content, ok := result["content"].([]any)
	assert.True(t, ok)
	assert.Len(t, content, 1)

	item, ok := content[0].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "json", item["type"])
	assert.NotNil(t, item["json"])

	cancel()
	_ = outR.Close()
	_ = outW.Close()
	<-done
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}
