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
		sp := api.Space{
			ID:          "payments",
			Name:        "Payments",
			Description: "Payment services",
			QBE:         api.SpaceQuery{"domain:payments"},
		}

		w := spaceRequest(t, spaceRequestArgs{
			handler: router,
			method:  http.MethodPost,
			path:    "/engine/spaces",
			body:    sp,
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
		assert.Equal(t, sp.QBE, got.QBE)
		assert.Equal(t, &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$["tags"]["domain:payments"]`,
		}, got.Selector)

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
			QBE:         api.SpaceQuery{"domain:payments"},
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
		matching.Tags = api.Tags{"domain:payments"}
		excluded := helpers.NewSimpleStep("excluded")
		excluded.Tags = api.Tags{"domain:orders"}
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

func TestSpacePreviewAPI(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		matching := helpers.NewSimpleStep("matching")
		matching.Tags = api.Tags{"domain:payments"}
		excluded := helpers.NewSimpleStep("excluded")
		excluded.Tags = api.Tags{"domain:orders"}
		assert.NoError(t, env.Engine.RegisterStep(matching))
		assert.NoError(t, env.Engine.RegisterStep(excluded))

		// A Space being drafted has no ID or Name yet
		preview := func(sp api.Space) *httptest.ResponseRecorder {
			return spaceRequest(t, spaceRequestArgs{
				handler: env.Server.SetupRoutes(),
				method:  http.MethodPost,
				path:    "/engine/spaces/preview",
				body:    sp,
			})
		}

		w := preview(api.Space{Selector: &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$["tags"]["domain:payments"]`,
		}})
		assert.Equal(t, http.StatusOK, w.Code)
		var got api.SpacePreviewResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		assert.Equal(t, []api.StepID{matching.ID}, got.StepIDs)
		assert.Equal(t, api.ScriptLangJPath, got.Space.Selector.Language)

		w = preview(api.Space{
			QBE: api.SpaceQuery{"domain:payments"},
		})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		assert.Equal(t, []api.StepID{matching.ID}, got.StepIDs)
		assert.Equal(t, &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$["tags"]["domain:payments"]`,
		}, got.Space.Selector)

		w = preview(api.Space{Selector: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   `return value["tags"]["nothing"]`,
		}})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		assert.Empty(t, got.StepIDs)

		w = preview(api.Space{Selector: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "return value[",
		}})
		assert.Equal(t, http.StatusBadRequest, w.Code)
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

		w = startSpaceFlow(t, router, api.CreateFlowRequest{
			ID:      "space-flow",
			Goals:   []api.StepID{goal.ID},
			SpaceID: "payments",
		})
		assert.Equal(t, http.StatusCreated, w.Code)
		selected, err := env.Engine.GetFlowState("space-flow")
		assert.NoError(t, err)
		assert.Contains(t, selected.Plan.Steps, provider.ID)
	})
}

func TestStartFlowSpaceExcluded(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		steps := spaceSteps()
		provider := steps.provider
		goal := steps.goal
		provider.Tags = api.Tags{"domain:orders"}
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
		goal.Tags = append(goal.Tags, "environment:stage")
		provider.Tags = append(provider.Tags, "environment:production")
		assert.NoError(t, env.Engine.RegisterStep(goal))
		sp := paymentSpace()
		assert.NoError(t, env.Engine.RegisterSpace(sp))
		router := env.Server.SetupRoutes()

		w := startSpaceFlow(t, router, api.CreateFlowRequest{
			ID:      "before-step",
			Goals:   []api.StepID{goal.ID},
			SpaceID: sp.ID,
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)

		assert.NoError(t, env.Engine.RegisterStep(provider))
		w = startSpaceFlow(t, router, api.CreateFlowRequest{
			ID:      "after-step",
			Goals:   []api.StepID{goal.ID},
			SpaceID: sp.ID,
		})
		assert.Equal(t, http.StatusCreated, w.Code)
		started, err := env.Engine.GetFlowState("after-step")
		assert.NoError(t, err)
		assert.Contains(t, started.Plan.Steps, provider.ID)

		updated := api.Space{
			ID:   "payments",
			Name: "Payments",
			QBE:  api.SpaceQuery{"domain:payments", "environment:stage"},
		}
		assert.NoError(t, env.Engine.UpdateSpace(updated))
		w = startSpaceFlow(t, router, api.CreateFlowRequest{
			ID:      "after-update",
			Goals:   []api.StepID{goal.ID},
			SpaceID: sp.ID,
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
		outside.Tags = api.Tags{"domain:orders"}
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
		QBE:  api.SpaceQuery{"domain:payments"},
	}
}

type spaceStepsRes struct {
	provider *api.Step
	goal     *api.Step
}

func spaceSteps() spaceStepsRes {
	provider := helpers.NewSimpleStep("provider")
	provider.Tags = api.Tags{"domain:payments"}
	provider.Attributes = api.AttributeSpecs{
		"data": {Role: api.RoleOutput, Type: api.TypeString},
	}
	goal := helpers.NewSimpleStep("goal")
	goal.Tags = api.Tags{"domain:payments"}
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
