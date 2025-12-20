package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestListStepsEmpty(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var response api.StepsListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Count)
	assert.Empty(t, response.Steps)
}

func TestDeleteStepSuccessful(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("to-delete")
	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/engine/step/to-delete", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var response api.MessageResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "unregistered")
}

func TestGetStepExists(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("existing-step")
	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine/step/existing-step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var retrieved api.Step
	err = json.Unmarshal(w.Body.Bytes(), &retrieved)
	assert.NoError(t, err)
	assert.Equal(t, api.StepID("existing-step"), retrieved.ID)
}

func TestListStepsWithMultiple(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step1 := helpers.NewSimpleStep("step1")
	step2 := helpers.NewSimpleStep("step2")
	step3 := helpers.NewSimpleStep("step3")

	err := env.Engine.RegisterStep(context.Background(), step1)
	assert.NoError(t, err)
	err = env.Engine.RegisterStep(context.Background(), step2)
	assert.NoError(t, err)
	err = env.Engine.RegisterStep(context.Background(), step3)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine/step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var response api.StepsListResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 3, response.Count)
	assert.Len(t, response.Steps, 3)
}

func TestCreateStepConflict(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("conflict-step")
	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	differentStep := &api.Step{
		ID:   "conflict-step",
		Name: "Different Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://different:8080",
		},
		Attributes: api.AttributeSpecs{},
	}

	body, _ := json.Marshal(differentStep)
	req := httptest.NewRequest("POST", "/engine/step", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, 409, w.Code)
}
