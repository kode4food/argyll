package server_test

import (
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
