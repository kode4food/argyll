package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
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

func TestCreateStep(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("test-step")

	body, _ := json.Marshal(step)
	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateStepIdempotent(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("dupe-step")
	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	body, _ := json.Marshal(step)
	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateStepInvalidBody(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader([]byte("not-json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateStepValid(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	body, _ := json.Marshal(&api.Step{})
	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListSteps(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("list-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine/step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.StepsListResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 1, response.Count)
	assert.Len(t, response.Steps, 1)
	assert.Equal(t, api.StepID("list-step"), response.Steps[0].ID)
}

func TestGetStep(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("get-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine/step/get-step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var retrieved *api.Step
	err = json.Unmarshal(w.Body.Bytes(), &retrieved)
	assert.NoError(t, err)
	assert.Equal(t, step.ID, retrieved.ID)
}

func TestDeleteStep(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("delete-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/engine/step/delete-step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteStepNotFound(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("DELETE", "/engine/step/missing-step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response api.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "not found")
}

func TestUpdateStep(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("update-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	updatedStep := helpers.NewSimpleStep("update-step")

	body, _ := json.Marshal(updatedStep)
	req := httptest.NewRequest(
		"PUT", "/engine/step/update-step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateStepMismatchStatus(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("update-step-mismatch")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	updatedStep := helpers.NewSimpleStep("other-id")

	body, _ := json.Marshal(updatedStep)
	req := httptest.NewRequest(
		"PUT", "/engine/step/update-step-mismatch", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetStepNotFound(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/step/nonexistent", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteStepMissing(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("DELETE", "/engine/step/nonexistent", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response api.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "not found")
}

func TestCreateStepInvalidRequest(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader([]byte("invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateStepValidBody(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := &api.Step{
		ID:   "",
		Name: "Invalid Step",
		Type: api.StepTypeSync,
	}

	body, _ := json.Marshal(step)
	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateStepMismatchMessage(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("original-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	updatedStep := helpers.NewSimpleStep("different-id")

	body, _ := json.Marshal(updatedStep)
	req := httptest.NewRequest(
		"PUT", "/engine/step/original-step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "does not match")
}

func TestUpdateValidationError(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("update-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	invalidStep := &api.Step{
		ID:   "update-step",
		Name: "",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{},
	}

	body, _ := json.Marshal(invalidStep)
	req := httptest.NewRequest(
		"PUT", "/engine/step/update-step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateStepNotFound(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("nonexistent")

	body, _ := json.Marshal(step)
	req := httptest.NewRequest(
		"PUT", "/engine/step/nonexistent", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}

func TestUpdateStepInvalidJSON(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest(
		"PUT",
		"/engine/step/test-step",
		bytes.NewReader([]byte("invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateStepDuplicate(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("duplicate-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	body, _ := json.Marshal(step)
	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateStepInvalidScript(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewScriptStep(
		"invalid-script", api.ScriptLangLua, "return {invalid syntax", "result",
	)

	body, _ := json.Marshal(step)
	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to register step")
}

func TestCreateStepInvalidText(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader([]byte("not json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteStepInternalError(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("test-delete-step")
	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/engine/step/test-delete-step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListStepsRunning(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step1 := helpers.NewSimpleStep("step-1")
	step2 := helpers.NewSimpleStep("step-2")

	err := env.Engine.RegisterStep(context.Background(), step1)
	assert.NoError(t, err)
	err = env.Engine.RegisterStep(context.Background(), step2)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine/step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.StepsListResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 2, response.Count)
}
