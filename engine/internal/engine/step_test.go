package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	as "github.com/kode4food/spuds/engine/internal/assert"
	"github.com/kode4food/spuds/engine/internal/assert/helpers"
	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestGetActiveFlow(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	ctx := context.Background()
	step := helpers.NewSimpleStep("active-test")

	err := env.Engine.RegisterStep(ctx, step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"active-test"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		ctx, "wf-active-test", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	flow, err := env.Engine.GetFlowState(ctx, "wf-active-test")
	require.NoError(t, err)
	assert.NotNil(t, flow)
	assert.Equal(t, api.FlowID("wf-active-test"), flow.ID)
	assert.Equal(t, api.FlowActive, flow.Status)
}

func TestGetFlowNotFound(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	ctx := context.Background()
	_, err := env.Engine.GetFlowState(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "flow not found")
}

func TestScript(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := &api.Step{
		ID:   "script-step",
		Name: "Script Step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangAle,
			Script:   `{:result "success"}`,
		},
		Attributes: map[api.Name]*api.AttributeSpec{
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"script-step"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(),
		"wf-script",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	a := as.New(t)
	fs := engine.FlowStep{FlowID: "wf-script", StepID: "script-step"}
	a.EventuallyWithError(func() error {
		_, err := env.Engine.GetCompiledScript(fs)
		return err
	}, 500*time.Millisecond, "script should compile")

	comp, err := env.Engine.GetCompiledScript(fs)
	require.NoError(t, err)
	assert.NotNil(t, comp)
}

func TestScriptMissing(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("no-script")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"no-script"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(),
		"wf-no-script",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	fs := engine.FlowStep{FlowID: "wf-no-script", StepID: "no-script"}
	_, err = env.Engine.GetCompiledScript(fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution plan missing")
}

func TestPredicate(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := helpers.NewStepWithPredicate(
		"predicate-step", api.ScriptLangLua, "return true",
	)

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"predicate-step"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(),
		"wf-predicate",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	a := as.New(t)
	fs := engine.FlowStep{FlowID: "wf-predicate", StepID: "predicate-step"}
	a.EventuallyWithError(func() error {
		_, err := env.Engine.GetCompiledPredicate(fs)
		return err
	}, 500*time.Millisecond, "predicate should compile")

	comp, err := env.Engine.GetCompiledPredicate(fs)
	require.NoError(t, err)
	assert.NotNil(t, comp)
}

func TestPlanFlowNotFound(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	fs := engine.FlowStep{FlowID: "nonexistent-flow", StepID: "step-id"}
	_, err := env.Engine.GetCompiledScript(fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "flow not found")
}
