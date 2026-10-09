package mcp_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp"
)

func TestNewServerTrimsTrailingSlash(t *testing.T) {
	hc := &http.Client{
		Transport: roundTripperFunc(
			func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "/engine/steps", r.URL.Path)
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
