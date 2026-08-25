package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestSpaceAPI(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		router := env.Server.SetupRoutes()
		space := api.Space{
			ID:          "payments",
			Name:        "Payments",
			Description: "Payment services",
			Selector: api.LabelSelector{MatchLabels: api.Labels{
				"domain": "payments",
			}},
		}

		w := spaceRequest(t, spaceRequestArgs{
			handler: router,
			method:  http.MethodPost,
			path:    "/engine/spaces",
			body:    space,
		})
		assert.Equal(t, http.StatusCreated, w.Code)

		w = spaceRequest(t, spaceRequestArgs{
			handler: router,
			method:  http.MethodGet,
			path:    "/engine/spaces/payments",
		})
		assert.Equal(t, http.StatusOK, w.Code)
		var got api.Space
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		assert.Equal(t, space, got)

		w = spaceRequest(t, spaceRequestArgs{
			handler: router,
			method:  http.MethodGet,
			path:    "/engine/spaces",
		})
		assert.Equal(t, http.StatusOK, w.Code)
		var listed api.SpacesListResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &listed))
		assert.Equal(t, 1, listed.Count)

		updated := api.Space{
			ID:          "payments",
			Name:        "Payments",
			Description: "Updated",
			Selector: api.LabelSelector{MatchLabels: api.Labels{
				"domain": "payments",
			}},
		}
		w = spaceRequest(t, spaceRequestArgs{
			handler: router,
			method:  http.MethodPut,
			path:    "/engine/spaces/payments",
			body:    updated,
		})
		assert.Equal(t, http.StatusOK, w.Code)

		w = spaceRequest(t, spaceRequestArgs{
			handler: router,
			method:  http.MethodDelete,
			path:    "/engine/spaces/payments",
		})
		assert.Equal(t, http.StatusOK, w.Code)

		w = spaceRequest(t, spaceRequestArgs{
			handler: router,
			method:  http.MethodGet,
			path:    "/engine/spaces/payments",
		})
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSpaceStepsAPI(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		matching := helpers.NewSimpleStep("matching")
		matching.Labels = api.Labels{"domain": "payments"}
		excluded := helpers.NewSimpleStep("excluded")
		excluded.Labels = api.Labels{"domain": "orders"}
		assert.NoError(t, env.Engine.RegisterStep(matching))
		assert.NoError(t, env.Engine.RegisterStep(excluded))
		assert.NoError(t, env.Engine.RegisterSpace(paymentSpace()))
		router := env.Server.SetupRoutes()

		w := spaceRequest(t, spaceRequestArgs{
			handler: router,
			method:  http.MethodGet,
			path:    "/engine/spaces/payments/steps",
		})
		assert.Equal(t, http.StatusOK, w.Code)
		var got api.StepsListResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		assert.Equal(t, 1, got.Count)
		if assert.Len(t, got.Steps, 1) {
			assert.Equal(t, matching.ID, got.Steps[0].ID)
		}

		w = spaceRequest(t, spaceRequestArgs{
			handler: router,
			method:  http.MethodGet,
			path:    "/engine/spaces/missing/steps",
		})
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestStartFlowSpace(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		steps := spaceSteps()
		provider := steps.provider
		goal := steps.goal
		assert.NoError(t, env.Engine.RegisterStep(provider))
		assert.NoError(t, env.Engine.RegisterStep(goal))
		assert.NoError(t, env.Engine.RegisterSpace(paymentSpace()))
		router := env.Server.SetupRoutes()

		w := startSpaceFlow(t, router, api.CreateFlowRequest{
			ID:    "global-flow",
			Goals: []api.StepID{goal.ID},
		})
		assert.Equal(t, http.StatusCreated, w.Code)
		global, err := env.Engine.GetFlowState("global-flow")
		assert.NoError(t, err)
		assert.Contains(t, global.Plan.Steps, provider.ID)
		assert.Empty(t, global.SpaceID)

		w = startSpaceFlow(t, router, api.CreateFlowRequest{
			ID:      "space-flow",
			Goals:   []api.StepID{goal.ID},
			SpaceID: "payments",
		})
		assert.Equal(t, http.StatusCreated, w.Code)
		selected, err := env.Engine.GetFlowState("space-flow")
		assert.NoError(t, err)
		assert.Contains(t, selected.Plan.Steps, provider.ID)
		assert.Equal(t, api.SpaceID("payments"), selected.SpaceID)
	})
}

func TestStartFlowSpaceExcluded(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		steps := spaceSteps()
		provider := steps.provider
		goal := steps.goal
		provider.Labels["domain"] = "orders"
		assert.NoError(t, env.Engine.RegisterStep(provider))
		assert.NoError(t, env.Engine.RegisterStep(goal))
		assert.NoError(t, env.Engine.RegisterSpace(paymentSpace()))
		router := env.Server.SetupRoutes()

		w := startSpaceFlow(t, router, api.CreateFlowRequest{
			ID:      "excluded-flow",
			Goals:   []api.StepID{goal.ID},
			SpaceID: "payments",
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestStartFlowUnknownSpace(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		st := helpers.NewSimpleStep("goal")
		assert.NoError(t, env.Engine.RegisterStep(st))
		w := startSpaceFlow(t, env.Server.SetupRoutes(),
			api.CreateFlowRequest{
				ID:      "unknown-space",
				Goals:   []api.StepID{st.ID},
				SpaceID: "missing",
			})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSpaceDynamicPlanning(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		steps := spaceSteps()
		provider := steps.provider
		goal := steps.goal
		goal.Labels["environment"] = "stage"
		provider.Labels["environment"] = "production"
		assert.NoError(t, env.Engine.RegisterStep(goal))
		space := paymentSpace()
		assert.NoError(t, env.Engine.RegisterSpace(space))
		router := env.Server.SetupRoutes()

		w := startSpaceFlow(t, router, api.CreateFlowRequest{
			ID:      "before-step",
			Goals:   []api.StepID{goal.ID},
			SpaceID: space.ID,
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)

		assert.NoError(t, env.Engine.RegisterStep(provider))
		w = startSpaceFlow(t, router, api.CreateFlowRequest{
			ID:      "after-step",
			Goals:   []api.StepID{goal.ID},
			SpaceID: space.ID,
		})
		assert.Equal(t, http.StatusCreated, w.Code)
		started, err := env.Engine.GetFlowState("after-step")
		assert.NoError(t, err)
		assert.Contains(t, started.Plan.Steps, provider.ID)

		updated := api.Space{
			ID:   "payments",
			Name: "Payments",
			Selector: api.LabelSelector{MatchLabels: api.Labels{
				"domain":      "payments",
				"environment": "stage",
			}},
		}
		assert.NoError(t, env.Engine.UpdateSpace(updated))
		w = startSpaceFlow(t, router, api.CreateFlowRequest{
			ID:      "after-update",
			Goals:   []api.StepID{goal.ID},
			SpaceID: space.ID,
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)

		unchanged, err := env.Engine.GetFlowState("after-step")
		assert.NoError(t, err)
		assert.Contains(t, unchanged.Plan.Steps, provider.ID)
	})
}

func TestPlanPreviewSpace(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		steps := spaceSteps()
		outside := helpers.NewSimpleStep("outside")
		outside.Labels = api.Labels{"domain": "orders"}
		outside.Attributes = api.AttributeSpecs{
			"data": {Role: api.RoleOutput, Type: api.TypeString},
		}
		assert.NoError(t, env.Engine.RegisterStep(steps.provider))
		assert.NoError(t, env.Engine.RegisterStep(steps.goal))
		assert.NoError(t, env.Engine.RegisterStep(outside))
		assert.NoError(t, env.Engine.RegisterSpace(paymentSpace()))
		router := env.Server.SetupRoutes()

		w := previewSpacePlan(t, router, api.ExecutionPlanRequest{
			Goals:   []api.StepID{steps.goal.ID},
			SpaceID: "payments",
		})
		assert.Equal(t, http.StatusOK, w.Code)
		var scoped api.ExecutionPlan
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &scoped))
		assert.Contains(t, scoped.Steps, steps.provider.ID)
		assert.NotContains(t, scoped.Steps, outside.ID)

		w = previewSpacePlan(t, router, api.ExecutionPlanRequest{
			Goals: []api.StepID{steps.goal.ID},
		})
		assert.Equal(t, http.StatusOK, w.Code)
		var global api.ExecutionPlan
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &global))
		assert.Contains(t, global.Steps, outside.ID)

		w = previewSpacePlan(t, router, api.ExecutionPlanRequest{
			Goals:   []api.StepID{steps.goal.ID},
			SpaceID: "missing",
		})
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func previewSpacePlan(
	t *testing.T, handler http.Handler, req api.ExecutionPlanRequest,
) *httptest.ResponseRecorder {
	t.Helper()
	return spaceRequest(t, spaceRequestArgs{
		handler: handler,
		method:  http.MethodPost,
		path:    "/engine/plan",
		body:    req,
	})
}

func paymentSpace() api.Space {
	return api.Space{
		ID:   "payments",
		Name: "Payments",
		Selector: api.LabelSelector{MatchLabels: api.Labels{
			"domain": "payments",
		}},
	}
}

type spaceStepsRes struct {
	provider *api.Step
	goal     *api.Step
}

func spaceSteps() spaceStepsRes {
	provider := helpers.NewSimpleStep("provider")
	provider.Labels = api.Labels{"domain": "payments"}
	provider.Attributes = api.AttributeSpecs{
		"data": {Role: api.RoleOutput, Type: api.TypeString},
	}
	goal := helpers.NewSimpleStep("goal")
	goal.Labels = api.Labels{"domain": "payments"}
	goal.Attributes = api.AttributeSpecs{
		"data":   {Role: api.RoleRequired, Type: api.TypeString},
		"result": {Role: api.RoleOutput, Type: api.TypeString},
	}
	return spaceStepsRes{provider: provider, goal: goal}
}

type spaceRequestArgs struct {
	handler http.Handler
	method  string
	path    string
	body    any
}

func spaceRequest(
	t *testing.T, args spaceRequestArgs,
) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if args.body != nil {
		var err error
		data, err = json.Marshal(args.body)
		assert.NoError(t, err)
	}
	req := httptest.NewRequest(args.method, args.path, bytes.NewReader(data))
	req.Header.Set("Content-Type", api.JSONContentType)
	w := httptest.NewRecorder()
	args.handler.ServeHTTP(w, req)
	return w
}

func startSpaceFlow(
	t *testing.T, handler http.Handler, req api.CreateFlowRequest,
) *httptest.ResponseRecorder {
	t.Helper()
	return spaceRequest(t, spaceRequestArgs{
		handler: handler,
		method:  http.MethodPost,
		path:    "/engine/flows",
		body:    req,
	})
}
