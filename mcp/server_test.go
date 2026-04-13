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
