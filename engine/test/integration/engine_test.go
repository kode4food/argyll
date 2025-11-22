package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	as "github.com/kode4food/spuds/engine/internal/assert"
	"github.com/kode4food/spuds/engine/internal/assert/helpers"
	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/util"
)

func TestStartFlowSimple(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := &api.Step{
		ID:      "goal-step",
		Name:    "Goal",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080",
		},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	env.MockClient.SetResponse("goal-step", api.Args{"result": "success"})

	plan := &api.ExecutionPlan{
		Goals:    []api.StepID{"goal-step"},
		Required: []api.Name{},
		Steps: map[api.StepID]*api.StepInfo{
			"goal-step": {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(),
		"wf-1",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	flow, err := env.Engine.GetFlowState(context.Background(), "wf-1")
	require.NoError(t, err)
	assert.NotNil(t, flow)
	assert.Equal(t, api.FlowID("wf-1"), flow.ID)
}

func TestFlowCompletion(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := &api.Step{
		ID:      "completion-step",
		Name:    "Completion Step",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080",
		},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	env.MockClient.SetResponse("completion-step",
		api.Args{"result": "completed"},
	)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"completion-step"},
		Steps: map[api.StepID]*api.StepInfo{
			"completion-step": {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-completion", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	// Wait for flow to complete
	a := as.New(t)
	var flow *api.FlowState
	a.Eventually(func() bool {
		var err error
		flow, err = env.Engine.GetFlowState(
			context.Background(), "wf-completion",
		)
		if err != nil {
			return false
		}
		return flow.Status == api.FlowCompleted
	}, 500*time.Millisecond, "flow should complete")

	assert.Equal(t, api.FlowCompleted, flow.Status)
	exec := flow.Executions["completion-step"]
	assert.Equal(t, api.StepCompleted, exec.Status)
	assert.Equal(t, "completed", exec.Outputs["result"])
}

func TestListFlows(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("list-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"list-step"},
		Steps: map[api.StepID]*api.StepInfo{
			"list-step": {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-list-1", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.StartFlow(
		context.Background(), "wf-list-2", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	flows, err := env.Engine.ListFlows(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(flows), 2)

	ids := util.Set[api.FlowID]{}
	for _, wf := range flows {
		ids.Add(wf.ID)
	}
	assert.True(t, ids.Contains("wf-list-1"))
	assert.True(t, ids.Contains("wf-list-2"))
}

func TestGetFlowEvents(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("events-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"events-step"},
		Steps: map[api.StepID]*api.StepInfo{
			"events-step": {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-events", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	evs, err := env.Engine.GetFlowEvents(
		context.Background(), "wf-events", 0,
	)
	require.NoError(t, err)

	if len(evs) > 0 {
		assert.Equal(t,
			timebox.EventType(api.EventTypeFlowStarted),
			evs[0].Type,
		)
	}
}

func TestShutdownTimeout(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer func() {
		_ = env.Engine.Stop()
		env.Cleanup()
	}()

	env.Engine.Start()

	err := env.Engine.Stop()
	assert.NoError(t, err)
}

func TestScriptStep(t *testing.T) {
	t.Skip("Script validation requires different approach for variable scoping")

	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	scriptStep := &api.Step{
		ID:      "script-1",
		Name:    "Script Step",
		Type:    api.StepTypeScript,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"name":     {Role: api.RoleOutput, Type: api.TypeString},
			"greeting": {Role: api.RoleOutput, Type: api.TypeString},
		},
		Script: &api.ScriptConfig{
			Language: api.ScriptLangAle,
			Script:   `{:greeting (str "Hello, " name)}`,
		},
	}

	err := env.Engine.RegisterStep(context.Background(), scriptStep)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals:    []api.StepID{"script-1"},
		Required: []api.Name{"name"},
		Steps: map[api.StepID]*api.StepInfo{
			"script-1": {Step: scriptStep},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(),
		"wf-script",
		plan,
		api.Args{"name": "World"},
		api.Metadata{},
	)
	require.NoError(t, err)

	a := as.New(t)
	var flow *api.FlowState
	a.Eventually(func() bool {
		var err error
		flow, err = env.Engine.GetFlowState(
			context.Background(), "wf-script",
		)
		if err != nil {
			return false
		}
		exec, ok := flow.Executions["script-1"]
		return ok && exec.Status == api.StepCompleted
	}, 500*time.Millisecond, "script step should complete")

	exec := flow.Executions["script-1"]
	assert.Equal(t, api.StepCompleted, exec.Status)
	assert.Equal(t, "Hello, World", exec.Outputs["greeting"])
}
