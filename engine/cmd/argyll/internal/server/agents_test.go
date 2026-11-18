package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/cmd/argyll/internal/server"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

func TestAgentCapabilities(t *testing.T) {
	anthropic := api.AgentProvider{
		Provider: "anthropic",
		Models:   []string{"claude"},
		MCP:      []api.MCPServer{{Server: "crm", Tools: []string{"lookup"}}},
	}
	local := api.AgentProvider{
		Provider: "local",
		Models:   []string{"model"},
		MCP:      []api.MCPServer{},
	}
	var calls atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.Equal(t, builtins.AgentCapabilitiesPath, r.URL.Path)
		_ = json.NewEncoder(w).Encode(api.AgentCapabilities{
			Providers: []api.AgentProvider{anthropic, {Provider: "unrouted"}},
		})
	}))
	defer primary.Close()
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, builtins.AgentCapabilitiesPath, r.URL.Path)
		_ = json.NewEncoder(w).Encode(api.AgentCapabilities{
			Providers: []api.AgentProvider{local, {Provider: "anthropic"}, {Provider: "down"}},
		})
	}))
	defer fallback.Close()
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer down.Close()

	t.Run("combines routable providers", func(t *testing.T) {
		w := getAgentCapabilities(builtins.AgentWorkers{
			"anthropic": primary.URL,
			"same":      primary.URL,
			"down":      down.URL,
			"*":         fallback.URL,
		})
		assert.Equal(t, http.StatusOK, w.Code)
		var res api.AgentCapabilities
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		assert.Equal(t, []api.AgentProvider{anthropic, local}, res.Providers)
		assert.EqualValues(t, 1, calls.Load())
	})

	t.Run("no workers", func(t *testing.T) {
		w := getAgentCapabilities(nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"providers":[]}`, w.Body.String())
	})

	t.Run("all workers unavailable", func(t *testing.T) {
		w := getAgentCapabilities(builtins.AgentWorkers{"*": down.URL})
		assert.Equal(t, http.StatusBadGateway, w.Code)
	})
}

func getAgentCapabilities(workers builtins.AgentWorkers) *httptest.ResponseRecorder {
	router := gin.New()
	server.RegisterAgentRoutes(router, workers)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/engine/agents/capabilities", nil)
	router.ServeHTTP(w, req)
	return w
}
