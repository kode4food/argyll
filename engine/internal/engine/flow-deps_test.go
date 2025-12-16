package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestNothing(t *testing.T) {}

func TestNoDeps(t *testing.T) {
	e := &engine.Engine{}
	flow := &api.FlowState{
		Plan: &api.ExecutionPlan{
			Attributes: api.AttributeGraph{},
		},
	}

	assert.False(t, e.HasInputProvider("missing", flow))
}

func TestNoProviders(t *testing.T) {
	e := &engine.Engine{}
	flow := &api.FlowState{
		Plan: &api.ExecutionPlan{
			Attributes: api.AttributeGraph{
				"input": {Providers: []api.StepID{}},
			},
		},
	}

	assert.True(t, e.HasInputProvider("input", flow))
}

func TestCompletableProvider(t *testing.T) {
	e := &engine.Engine{}
	flow := &api.FlowState{
		Plan: &api.ExecutionPlan{
			Attributes: api.AttributeGraph{
				"input": {Providers: []api.StepID{"provider"}},
			},
			Steps: api.Steps{
				"provider": {
					ID:   "provider",
					Name: "Provider",
					Type: api.StepTypeSync,
					Attributes: api.AttributeSpecs{
						"input": {Role: api.RoleOutput, Type: api.TypeString},
					},
				},
			},
		},
		Executions: api.Executions{
			"provider": {Status: api.StepCompleted},
		},
	}

	assert.True(t, e.HasInputProvider("input", flow))
}

func TestFailedProvider(t *testing.T) {
	e := &engine.Engine{}
	flow := &api.FlowState{
		Plan: &api.ExecutionPlan{
			Attributes: api.AttributeGraph{
				"input": {Providers: []api.StepID{"provider"}},
			},
			Steps: api.Steps{
				"provider": {
					ID:   "provider",
					Name: "Provider",
					Type: api.StepTypeSync,
					Attributes: api.AttributeSpecs{
						"input": {Role: api.RoleOutput, Type: api.TypeString},
					},
				},
			},
		},
		Executions: api.Executions{
			"provider": {Status: api.StepFailed},
		},
	}

	assert.False(t, e.HasInputProvider("input", flow))
}

func TestGoalBlocked(t *testing.T) {
	e := &engine.Engine{}

	flow := &api.FlowState{
		Plan: &api.ExecutionPlan{
			Goals: []api.StepID{"goal"},
			Steps: api.Steps{
				"goal": {
					ID:   "goal",
					Name: "Goal",
					Type: api.StepTypeSync,
					Attributes: api.AttributeSpecs{
						"required": {
							Role: api.RoleRequired,
							Type: api.TypeString,
						},
					},
				},
				"provider": {
					ID:   "provider",
					Name: "Provider",
					Type: api.StepTypeSync,
					Attributes: api.AttributeSpecs{
						"required": {
							Role: api.RoleOutput,
							Type: api.TypeString,
						},
					},
				},
			},
			Attributes: api.AttributeGraph{
				"required": {Providers: []api.StepID{"provider"}},
			},
		},
		Executions: api.Executions{
			"goal": {Status: api.StepPending},
			// provider failed, so required input cannot be satisfied
			"provider": {Status: api.StepFailed},
		},
	}

	assert.True(t, e.IsFlowFailed(flow))
}

func TestGoalCompleted(t *testing.T) {
	e := &engine.Engine{}

	flow := &api.FlowState{
		Plan: &api.ExecutionPlan{
			Goals: []api.StepID{"goal"},
			Steps: api.Steps{
				"goal": {
					ID:   "goal",
					Name: "Goal",
					Type: api.StepTypeSync,
				},
			},
		},
		Executions: api.Executions{
			"goal": {Status: api.StepCompleted},
		},
	}

	assert.False(t, e.IsFlowFailed(flow))
}
