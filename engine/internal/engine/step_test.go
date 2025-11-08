package engine_test

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
)

func TestGetActive(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("active-test")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		GoalSteps: []timebox.ID{"active-test"},
		Steps:     []*api.Step{step},
	}

	err = env.Engine.StartWorkflow(
		context.Background(),
		"wf-active-test",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	workflow, ok := env.Engine.GetActiveWorkflow("wf-active-test")
	assert.True(t, ok)
	assert.NotNil(t, workflow)
	assert.Equal(t, timebox.ID("wf-active-test"), workflow.ID)
}

func TestGetActiveNotFound(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	_, ok := env.Engine.GetActiveWorkflow("nonexistent")
	assert.False(t, ok)
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
		GoalSteps: []timebox.ID{"script-step"},
		Steps:     []*api.Step{step},
	}

	err = env.Engine.StartWorkflow(
		context.Background(),
		"wf-script",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	a := as.New(t)
	a.EventuallyWithError(func() error {
		_, err := env.Engine.GetCompiledScript("wf-script", "script-step")
		return err
	}, 500*time.Millisecond, "script should compile")

	comp, err := env.Engine.GetCompiledScript("wf-script", "script-step")
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
		GoalSteps: []timebox.ID{"no-script"},
		Steps:     []*api.Step{step},
	}

	err = env.Engine.StartWorkflow(
		context.Background(),
		"wf-no-script",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	_, err = env.Engine.GetCompiledScript("wf-no-script", "no-script")
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
		GoalSteps: []timebox.ID{"predicate-step"},
		Steps:     []*api.Step{step},
	}

	err = env.Engine.StartWorkflow(
		context.Background(),
		"wf-predicate",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	a := as.New(t)
	a.EventuallyWithError(func() error {
		_, err := env.Engine.GetCompiledPredicate(
			"wf-predicate", "predicate-step",
		)
		return err
	}, 500*time.Millisecond, "predicate should compile")

	comp, err := env.Engine.GetCompiledPredicate(
		"wf-predicate", "predicate-step",
	)
	require.NoError(t, err)
	assert.NotNil(t, comp)
}

func TestPlanWorkflowNotFound(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	_, err := env.Engine.GetCompiledScript("nonexistent-workflow", "step-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")
}
