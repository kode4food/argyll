package mcp_test

import (
	"cmp"
	"io"
	"net/http"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp"
)

func TestNewServerTrimsTrailingSlash(t *testing.T) {
	hc := &http.Client{
		Transport: roundTripperFunc(
			func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "/engine/steps", r.URL.Path)
				return jsonResponse(
					http.StatusOK, []byte(`{"steps":[]}`),
				), nil
			},
		),
	}
	c := newClient(t, mcp.NewServer("http://example/", hc))
	_ = callToolText(t, c, "list_steps", map[string]any{})
}

func TestServerHandler(t *testing.T) {
	h, err := mcp.NewServer("", nil).Handler()
	assert.NoError(t, err)
	assert.NotNil(t, h)
}

func TestEngineResponseFailures(t *testing.T) {
	for _, tc := range []struct {
		name    string
		baseURL string
		resp    func() *http.Response
	}{
		{
			name: "empty body",
			resp: func() *http.Response {
				return jsonResponse(http.StatusOK, []byte(" "))
			},
		},
		{
			name: "invalid JSON",
			resp: func() *http.Response {
				return jsonResponse(http.StatusOK, []byte("{"))
			},
		},
		{
			name: "unreadable body",
			resp: func() *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(iotest.ErrReader(io.ErrUnexpectedEOF)),
				}
			},
		},
		{
			name:    "invalid base URL",
			baseURL: "http://bad host",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hc := &http.Client{
				Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
					return tc.resp(), nil
				}),
			}
			c := newClient(t, mcp.NewServer(cmp.Or(tc.baseURL, "http://example"), hc))
			assert.True(t, callToolFails(t, c, "list_steps", map[string]any{}))
		})
	}
}
