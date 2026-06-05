package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type eventsResponse struct {
	Events []map[string]any `json:"events"`
	Count  int              `json:"count"`
}

func TestGetCatalog(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("catalog-step")
		assert.NoError(t, testEnv.Engine.RegisterStep(st))

		req := httptest.NewRequest("GET", "/engine/catalog", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var cat api.CatalogState
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &cat))
		assert.Contains(t, cat.Steps, api.StepID("catalog-step"))
	})
}

func TestGetCatalogEmpty(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/engine/catalog", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var cat api.CatalogState
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &cat))
		assert.Empty(t, cat.Steps)
	})
}

func TestGetCatalogClosed(t *testing.T) {
	env := helpers.NewTestEngine(t)
	srv := server.NewServer(env.Engine, env.EventHub)
	env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/catalog", nil)
	w := httptest.NewRecorder()

	router := srv.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetCatalogEvents(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("catalog-events-step")
		assert.NoError(t, testEnv.Engine.RegisterStep(st))

		req := httptest.NewRequest("GET", "/engine/catalog/events", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp eventsResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Greater(t, resp.Count, 0)
		assert.Len(t, resp.Events, resp.Count)
	})
}

func TestGetCatalogEventsEmpty(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/engine/catalog/events", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp eventsResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 0, resp.Count)
		assert.Empty(t, resp.Events)
	})
}

func TestGetCatalogEventsClosed(t *testing.T) {
	env := helpers.NewTestEngine(t)
	srv := server.NewServer(env.Engine, env.EventHub)
	env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/catalog/events", nil)
	w := httptest.NewRecorder()

	router := srv.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetCluster(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/engine/cluster", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var cluster api.ClusterState
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &cluster))
		localID := testEnv.Engine.LocalNodeID()
		assert.Contains(t, cluster.Nodes, localID)
	})
}

func TestGetClusterClosed(t *testing.T) {
	env := helpers.NewTestEngine(t)
	srv := server.NewServer(env.Engine, env.EventHub)
	env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/cluster", nil)
	w := httptest.NewRecorder()

	router := srv.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetClusterEvents(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("cluster-events-step")
		assert.NoError(t, testEnv.Engine.RegisterStep(st))
		assert.NoError(t, testEnv.Engine.UpdateStepHealth(
			st.ID, api.HealthHealthy, "",
		))

		req := httptest.NewRequest("GET", "/engine/cluster/events", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp eventsResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Greater(t, resp.Count, 0)
		assert.Len(t, resp.Events, resp.Count)
	})
}

func TestGetClusterEventsEmpty(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/engine/cluster/events", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp eventsResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 0, resp.Count)
		assert.Empty(t, resp.Events)
	})
}

func TestGetClusterEventsClosed(t *testing.T) {
	env := helpers.NewTestEngine(t)
	srv := server.NewServer(env.Engine, env.EventHub)
	env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/cluster/events", nil)
	w := httptest.NewRecorder()

	router := srv.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetFlowEvents(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("flow-events-step")
		assert.NoError(t, testEnv.Engine.RegisterStep(st))

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"flow-events-step"},
			Steps: api.Steps{"flow-events-step": st},
		}
		assert.NoError(t, testEnv.Engine.StartFlow("flow-events-id", pl))

		req := httptest.NewRequest(
			"GET", "/engine/flow/flow-events-id/events", nil,
		)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp eventsResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Greater(t, resp.Count, 0)
		assert.Len(t, resp.Events, resp.Count)
	})
}

func TestGetFlowEventsNotFound(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest(
			"GET", "/engine/flow/nonexistent-flow/events", nil,
		)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
