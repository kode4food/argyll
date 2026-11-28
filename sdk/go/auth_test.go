package argyll_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	argyll "github.com/kode4food/argyll/sdk/go"
)

func TestServiceAccountToken(t *testing.T) {
	tokens, issued := tokenEndpoint(t, int(time.Hour.Seconds()))
	engine, seen := engineRecorder(t)
	setServiceAccount(t, tokens.URL)

	c := argyll.NewClient(engine.URL, time.Second)
	listSteps(t, c)
	listSteps(t, c)

	assert.Equal(t, []string{"Bearer svc-1", "Bearer svc-1"}, *seen)
	assert.EqualValues(t, 1, issued.Load())
}

func TestServiceAccountTokenRefreshes(t *testing.T) {
	// A token this close to expiry is renewed before each request
	tokens, issued := tokenEndpoint(t, 1)
	engine, seen := engineRecorder(t)
	setServiceAccount(t, tokens.URL)

	c := argyll.NewClient(engine.URL, time.Second)
	listSteps(t, c)
	listSteps(t, c)

	assert.Equal(t, []string{"Bearer svc-1", "Bearer svc-2"}, *seen)
	assert.EqualValues(t, 2, issued.Load())
}

func TestWithTokenReplacesServiceAccount(t *testing.T) {
	tokens, issued := tokenEndpoint(t, int(time.Hour.Seconds()))
	engine, seen := engineRecorder(t)
	setServiceAccount(t, tokens.URL)

	listSteps(t, argyll.NewClient(engine.URL, time.Second).WithToken("explicit"))

	assert.Equal(t, []string{"Bearer explicit"}, *seen)
	assert.EqualValues(t, 0, issued.Load())
}

func TestWithTokenRedirects(t *testing.T) {
	target, seen := engineRecorder(t)
	var crossOrigin atomic.Bool
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirected" {
			assert.Equal(t, "Bearer explicit", r.Header.Get("Authorization"))
			_ = json.NewEncoder(w).Encode(api.StepsListResponse{})
			return
		}
		if crossOrigin.Load() {
			http.Redirect(w, r, target.URL, http.StatusFound)
			return
		}
		http.Redirect(w, r, "/redirected", http.StatusFound)
	}))
	t.Cleanup(gateway.Close)
	c := argyll.NewClient(gateway.URL, time.Second).WithToken("explicit")

	listSteps(t, c)

	crossOrigin.Store(true)
	_, err := c.ListSteps(context.Background())
	assert.ErrorIs(t, err, api.ErrCrossOriginRedirect)
	assert.Empty(t, *seen)
}

// tokenEndpoint issues numbered service-account tokens that expire after expiresIn seconds
func tokenEndpoint(t *testing.T, expiresIn int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var issued atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "client_credentials", r.PostForm.Get("grant_type"))
		id, secret, ok := r.BasicAuth()
		if !ok {
			id = r.PostForm.Get("client_id")
			secret = r.PostForm.Get("client_secret")
		}
		assert.Equal(t, "svc", id)
		assert.Equal(t, "shh", secret)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": fmt.Sprintf("svc-%d", issued.Add(1)),
			"token_type":   "Bearer",
			"expires_in":   expiresIn,
		})
	}))
	t.Cleanup(srv.Close)
	return srv, &issued
}

// engineRecorder records each request's Authorization header
func engineRecorder(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		_ = json.NewEncoder(w).Encode(api.StepsListResponse{})
	}))
	t.Cleanup(srv.Close)
	return srv, &seen
}

func setServiceAccount(t *testing.T, tokenURL string) {
	t.Helper()
	t.Setenv(argyll.TokenURLEnv, tokenURL)
	t.Setenv(argyll.ClientIDEnv, "svc")
	t.Setenv(argyll.ClientSecretEnv, "shh")
}

func listSteps(t *testing.T, c *argyll.Client) {
	t.Helper()
	_, err := c.ListSteps(context.Background())
	assert.NoError(t, err)
}
