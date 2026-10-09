package argyll_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	argyll "github.com/kode4food/argyll/sdk/go"
)

func TestNewFlow(t *testing.T) {
	client := argyll.NewClient("http://localhost:8080", 30*time.Second)
	wf := client.NewFlow("test-flow")
	assert.NotNil(t, wf)
}

func TestFlowWithGoals(t *testing.T) {
	goals := api.Goals{
		Steps: []api.StepID{"goal-1", "goal-2"},
		Else:  &api.Goals{Steps: []api.StepID{"goal-3"}},
	}
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)

			assert.Equal(t, api.FlowID("wf-1"), req.ID)
			assert.Equal(t, goals, req.Goals)

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := argyll.NewClient(server.URL, 5*time.Second)
	err := client.NewFlow("wf-1").
		WithGoals(goals).
		Start(context.Background())

	assert.NoError(t, err)
}

func TestFlowGoalsSnapshot(t *testing.T) {
	expected := api.Goals{
		Steps: []api.StepID{"primary"},
		Else:  &api.Goals{Steps: []api.StepID{"backup"}},
	}
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, expected, req.Goals)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	goals := api.Goals{
		Steps: []api.StepID{"primary"},
		Else:  &api.Goals{Steps: []api.StepID{"backup"}},
	}
	client := argyll.NewClient(server.URL, 5*time.Second)
	base := client.NewFlow("snapshot").WithGoals(goals)
	derived := base.WithGoal("extra")
	goals.Steps[0] = "changed-primary"
	goals.Else.Steps[0] = "changed-backup"
	goals.Else.Else = &api.Goals{Steps: []api.StepID{"changed-chain"}}

	assert.NoError(t, base.Start(context.Background()))
	expected.Else.Steps = append(expected.Else.Steps, "extra")
	assert.NoError(t, derived.Start(context.Background()))
}

func TestFlowWithGoal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)

			assert.Equal(t, api.FlowID("wf-1"), req.ID)
			assert.Equal(t, api.Goals{
				Steps: []api.StepID{"goal-1", "goal-2"},
				Else:  &api.Goals{Steps: []api.StepID{"goal-3", "goal-4"}},
			}, req.Goals)

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := argyll.NewClient(server.URL, 5*time.Second)
	err := client.NewFlow("wf-1").
		WithGoal("goal-1").
		WithGoal("goal-2").
		WithGoals(api.Goals{
			Steps: []api.StepID{"goal-1", "goal-2"},
			Else:  &api.Goals{Steps: []api.StepID{"goal-3"}},
		}).
		WithGoal("goal-4").
		Start(context.Background())

	assert.NoError(t, err)
}

func TestFlowWithInitialState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)

			assert.Equal(t, api.FlowID("wf-1"), req.ID)
			assert.Equal(t, []any{"value1"}, req.Init["key1"])
			assert.Equal(t, []any{float64(42)}, req.Init["key2"])

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := argyll.NewClient(server.URL, 5*time.Second)
	err := client.NewFlow("wf-1").
		WithGoals(api.Goals{Steps: []api.StepID{"goal-step"}}).
		WithInitialState(api.InitArgs{
			"key1": {"value1"},
			"key2": {42},
		}).
		Start(context.Background())

	assert.NoError(t, err)
}

func TestFlowWithTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)

			assert.Equal(t, api.FlowID("wf-1"), req.ID)
			assert.Equal(t, api.Tags{"env:dev", "team:core"}, req.Tags)

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := argyll.NewClient(server.URL, 5*time.Second)
	err := client.NewFlow("wf-1").
		WithGoals(api.Goals{Steps: []api.StepID{"goal-step"}}).
		WithTags("team:core").
		WithTags("env:dev").
		Start(context.Background())

	assert.NoError(t, err)
}

func TestFlowWithEmptyTags(t *testing.T) {
	wf := argyll.NewClient("http://localhost:8080", 30*time.Second).
		NewFlow("wf-1")

	assert.Equal(t, wf, wf.WithTags())
}

func TestFlowStartStatusCreated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		},
	))
	defer server.Close()

	client := argyll.NewClient(server.URL, 5*time.Second)
	err := client.NewFlow("wf-1").
		WithGoals(api.Goals{Steps: []api.StepID{"goal-step"}}).
		Start(context.Background())

	assert.NoError(t, err)
}

func TestFlowStartError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request"))
		},
	))
	defer server.Close()

	client := argyll.NewClient(server.URL, 5*time.Second)
	err := client.NewFlow("wf-1").
		WithGoals(api.Goals{Steps: []api.StepID{"goal-step"}}).
		Start(context.Background())

	assert.Error(t, err)
}

func TestFlowChaining(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)

			assert.Equal(t, api.FlowID("complex-flow"), req.ID)
			assert.Equal(t,
				api.Goals{Steps: []api.StepID{"goal-1", "goal-2"}}, req.Goals)
			assert.Equal(t, []any{"value1"}, req.Init["arg1"])
			assert.Equal(t, []any{float64(100)}, req.Init["arg2"])

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := argyll.NewClient(server.URL, 5*time.Second)
	err := client.NewFlow("complex-flow").
		WithGoals(api.Goals{Steps: []api.StepID{"goal-1", "goal-2"}}).
		WithInitialState(api.InitArgs{
			"arg1": {"value1"},
			"arg2": {100},
		}).
		Start(context.Background())

	assert.NoError(t, err)
}

func TestFlowImmutability(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Equal(t, api.Goals{Steps: []api.StepID{"goal-1"}}, req.Goals)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Equal(t, api.Goals{Steps: []api.StepID{"goal-2"}}, req.Goals)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server2.Close()

	client1 := argyll.NewClient(server1.URL, 5*time.Second)
	err1 := client1.NewFlow("base-wf").
		WithGoals(api.Goals{Steps: []api.StepID{"goal-1"}}).
		Start(context.Background())
	assert.NoError(t, err1)

	client2 := argyll.NewClient(server2.URL, 5*time.Second)
	err2 := client2.NewFlow("base-wf").
		WithGoals(api.Goals{Steps: []api.StepID{"goal-2"}}).
		Start(context.Background())
	assert.NoError(t, err2)
}

func TestImmutabilityInitState(t *testing.T) {
	initState1 := api.InitArgs{"key": {"value1"}}
	initState2 := api.InitArgs{"key": {"value2"}}

	server1 := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Equal(t, []any{"value1"}, req.Init["key"])
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Equal(t, []any{"value2"}, req.Init["key"])
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server2.Close()

	client1 := argyll.NewClient(server1.URL, 5*time.Second)
	err1 := client1.NewFlow("test-wf").
		WithGoals(api.Goals{Steps: []api.StepID{"goal"}}).
		WithInitialState(initState1).
		Start(context.Background())
	assert.NoError(t, err1)

	client2 := argyll.NewClient(server2.URL, 5*time.Second)
	err2 := client2.NewFlow("test-wf").
		WithGoals(api.Goals{Steps: []api.StepID{"goal"}}).
		WithInitialState(initState2).
		Start(context.Background())
	assert.NoError(t, err2)
}

func TestImmutabilityTags(t *testing.T) {
	tags1 := api.Tags{"team:core"}
	tags2 := api.Tags{"team:platform"}

	server1 := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Equal(t, tags1, req.Tags)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Equal(t, tags2, req.Tags)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server2.Close()

	client1 := argyll.NewClient(server1.URL, 5*time.Second)
	err1 := client1.NewFlow("test-wf").
		WithGoals(api.Goals{Steps: []api.StepID{"goal"}}).
		WithTags(tags1...).
		Start(context.Background())
	assert.NoError(t, err1)

	client2 := argyll.NewClient(server2.URL, 5*time.Second)
	err2 := client2.NewFlow("test-wf").
		WithGoals(api.Goals{Steps: []api.StepID{"goal"}}).
		WithTags(tags2...).
		Start(context.Background())
	assert.NoError(t, err2)
}

func TestFlowEmptyGoals(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Empty(t, req.Goals.Steps)
			assert.Nil(t, req.Goals.Else)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := argyll.NewClient(server.URL, 5*time.Second)
	err := client.NewFlow("wf-1").Start(context.Background())
	assert.NoError(t, err)
}

func TestFlowEmptyInitialState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Empty(t, req.Init)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := argyll.NewClient(server.URL, 5*time.Second)
	err := client.NewFlow("wf-1").
		WithGoals(api.Goals{Steps: []api.StepID{"goal-step"}}).
		Start(context.Background())
	assert.NoError(t, err)
}
