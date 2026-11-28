package mcp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/kode4food/argyll/mcp"
)

type (
	// headerTransport adds an Authorization header to every MCP request
	headerTransport struct {
		auth string
	}
)

func TestForwardsCallerAuthorization(t *testing.T) {
	var seen []string
	s := mcp.NewServer("http://engine", engineRecorder(&seen))
	h, err := s.Handler()
	if !assert.NoError(t, err) {
		return
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := sdk.NewClient(&sdk.Implementation{
		Name:    "test",
		Version: "1",
	}, nil)
	cs, err := client.Connect(context.Background(), &sdk.StreamableClientTransport{
		Endpoint:   srv.URL + mcp.HTTPPath,
		HTTPClient: &http.Client{Transport: headerTransport{auth: "Bearer user"}},
	}, nil)
	if !assert.NoError(t, err) {
		return
	}
	defer func() { _ = cs.Close() }()

	callToolText(t, cs, "list_steps", map[string]any{})
	assert.Equal(t, []string{"Bearer user"}, seen)
}

func TestHTTPCallerWithoutToken(t *testing.T) {
	var seen []string
	s := mcp.NewServer("http://engine", engineRecorder(&seen))
	h, err := s.Handler()
	if !assert.NoError(t, err) {
		return
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := sdk.NewClient(&sdk.Implementation{
		Name:    "test",
		Version: "1",
	}, nil)
	cs, err := client.Connect(context.Background(), &sdk.StreamableClientTransport{
		Endpoint: srv.URL + mcp.HTTPPath,
	}, nil)
	if !assert.NoError(t, err) {
		return
	}
	defer func() { _ = cs.Close() }()

	callToolText(t, cs, "list_steps", map[string]any{})
	assert.Equal(t, []string{""}, seen)
}

func TestHTTPReplicaSwitch(t *testing.T) {
	var seen []string
	s := mcp.NewServer("http://engine", engineRecorder(&seen))
	first, err := s.Handler()
	if !assert.NoError(t, err) {
		return
	}
	second, err := s.Handler()
	if !assert.NoError(t, err) {
		return
	}
	var replaced atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if replaced.Load() {
			second.ServeHTTP(w, r)
			return
		}
		first.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	client := sdk.NewClient(&sdk.Implementation{
		Name:    "test",
		Version: "1",
	}, nil)
	cs, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{
		Endpoint:             srv.URL + mcp.HTTPPath,
		HTTPClient:           &http.Client{Transport: headerTransport{auth: "Bearer user"}},
		DisableStandaloneSSE: true,
	}, nil)
	if !assert.NoError(t, err) {
		return
	}
	defer func() { _ = cs.Close() }()

	replaced.Store(true)
	callToolText(t, cs, "list_steps", map[string]any{})
	assert.Equal(t, []string{"Bearer user"}, seen)
}

func TestLocalClientUsesServiceTransport(t *testing.T) {
	var seen []string
	client := engineRecorder(&seen)
	client.Transport = &oauth2.Transport{
		Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "service"}),
		Base:   client.Transport,
	}
	s := mcp.NewServer("http://engine", client)
	callToolText(t, newClient(t, s), "list_steps", map[string]any{})
	assert.Equal(t, []string{"Bearer service"}, seen)
}

func (h headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", h.auth)
	return http.DefaultTransport.RoundTrip(r)
}

// engineRecorder answers every engine request and records its Authorization header
func engineRecorder(seen *[]string) *http.Client {
	return &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			*seen = append(*seen, r.Header.Get("Authorization"))
			return jsonResponse(http.StatusOK, []byte(`{"steps":[]}`)), nil
		}),
	}
}
