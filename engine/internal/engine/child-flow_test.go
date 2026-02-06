package engine_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type flowEvent struct {
	FlowID api.FlowID `json:"flow_id"`
}

const childFlowTimeout = 5 * time.Second

func TestFlowStepChildSuccess(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		child := &api.Step{
			ID:   "child-step",
			Name: "Child Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {}",
			},
			Attributes: api.AttributeSpecs{},
		}

		parent := &api.Step{
			ID:   "subflow-step",
			Name: "Subflow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{},
		}

		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		err := env.Engine.StartFlow("parent-flow", plan, api.Args{}, nil)
		assert.NoError(t, err)

		parentState := env.WaitForFlowStatus(t, "parent-flow", childFlowTimeout)
		assert.Equal(t, api.FlowCompleted, parentState.Status)

		exec := parentState.Executions[parent.ID]
		if assert.NotNil(t, exec) && assert.NotNil(t, exec.WorkItems) {
			var token api.Token
			for tkn := range exec.WorkItems {
				token = tkn
				break
			}

			childID := api.FlowID(fmt.Sprintf(
				"%s:%s:%s", "parent-flow", parent.ID, token,
			))
			childState, err := env.Engine.GetFlowState(childID)
			assert.NoError(t, err)
			assert.Equal(t, api.FlowCompleted, childState.Status)

			assert.Equal(t,
				api.FlowID("parent-flow"), metaFlowID(childState.Metadata),
			)
			assert.Equal(t, parent.ID, metaStepID(childState.Metadata))
			assert.Equal(t, token, metaToken(childState.Metadata))
		}
	})
}

func TestFlowStepChildFailureParentFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		child := &api.Step{
			ID:   "child-fail",
			Name: "Child Fail",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "error('boom')",
			},
			Attributes: api.AttributeSpecs{},
		}

		parent := &api.Step{
			ID:   "subflow-fail",
			Name: "Subflow Fail",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{},
		}

		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		err := env.Engine.StartFlow("parent-fail", plan, api.Args{}, nil)
		assert.NoError(t, err)

		parentState := env.WaitForFlowStatus(t, "parent-fail", childFlowTimeout)
		assert.Equal(t, api.FlowFailed, parentState.Status)
	})
}

func TestFlowStepMissingGoalParentFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		parent := &api.Step{
			ID:   "subflow-missing",
			Name: "Subflow Missing",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"missing-goal"},
			},
			Attributes: api.AttributeSpecs{},
		}

		assert.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		err := env.Engine.StartFlow("parent-missing", plan, api.Args{}, nil)
		assert.NoError(t, err)

		parentState := env.WaitForFlowStatus(t,
			"parent-missing", childFlowTimeout,
		)
		assert.Equal(t, api.FlowFailed, parentState.Status)
	})
}

func TestFlowStepMapping(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		child := &api.Step{
			ID:   "child-mapped",
			Name: "Child Mapped",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   "{:child_out child_in}",
			},
			Attributes: api.AttributeSpecs{
				"child_in":  {Role: api.RoleRequired},
				"child_out": {Role: api.RoleOutput},
			},
		}

		parent := &api.Step{
			ID:   "subflow-mapped",
			Name: "Subflow Mapped",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
				InputMap: map[api.Name]api.Name{
					"input": "child_in",
				},
				OutputMap: map[api.Name]api.Name{
					"child_out": "output",
				},
			},
			Attributes: api.AttributeSpecs{
				"input":  {Role: api.RoleRequired},
				"output": {Role: api.RoleOutput},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals:    []api.StepID{parent.ID},
			Steps:    api.Steps{parent.ID: parent},
			Required: []api.Name{"input"},
		}

		err := env.Engine.StartFlow(
			"parent-mapped", plan, api.Args{"input": float64(7)}, nil,
		)
		assert.NoError(t, err)

		parentState := env.WaitForFlowStatus(t,
			"parent-mapped", childFlowTimeout,
		)
		assert.Equal(t, api.FlowCompleted, parentState.Status)

		exec := parentState.Executions[parent.ID]
		if assert.NotNil(t, exec) {
			assert.Equal(t, float64(7), exec.Outputs["output"])
		}
	})
}

func TestListFlowsSkipsChildFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		consumer := env.EventHub.NewConsumer()
		defer consumer.Close()

		child := &api.Step{
			ID:   "child-list",
			Name: "Child List",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {}",
			},
			Attributes: api.AttributeSpecs{},
		}

		parent := &api.Step{
			ID:   "subflow-list",
			Name: "Subflow List",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{},
		}

		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		err := env.Engine.StartFlow("parent-list", plan, api.Args{}, nil)
		assert.NoError(t, err)

		parentState := env.WaitForFlowStatus(t, "parent-list", childFlowTimeout)
		assert.Equal(t, api.FlowCompleted, parentState.Status)

		exec := parentState.Executions[parent.ID]
		if !assert.NotNil(t, exec) {
			return
		}

		var token api.Token
		for tkn := range exec.WorkItems {
			token = tkn
			break
		}

		childID := api.FlowID(fmt.Sprintf(
			"%s:%s:%s", "parent-list", parent.ID, token,
		))

		waitForFlowEvents(t,
			consumer, childFlowTimeout, api.EventTypeFlowActivated,
			"parent-list", childID,
		)
		waitForFlowEvents(t,
			consumer, childFlowTimeout, api.EventTypeFlowDeactivated, childID,
		)

		flows, err := env.Engine.ListFlows()
		assert.NoError(t, err)

		var ids []api.FlowID
		for _, flow := range flows {
			ids = append(ids, flow.ID)
		}

		assert.Contains(t, ids, api.FlowID("parent-list"))
		assert.NotContains(t, ids, childID)
	})
}

func TestFlowStepMissingOutputParentFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		child := &api.Step{
			ID:   "child-empty",
			Name: "Child Empty",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {}",
			},
			Attributes: api.AttributeSpecs{
				"child_out": {Role: api.RoleOutput},
			},
		}

		parent := &api.Step{
			ID:   "subflow-missing-output",
			Name: "Subflow Missing Output",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
				OutputMap: map[api.Name]api.Name{
					"child_out": "output",
				},
			},
			Attributes: api.AttributeSpecs{
				"output": {Role: api.RoleOutput},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		err := env.Engine.StartFlow(
			"parent-missing-output", plan, api.Args{}, nil,
		)
		assert.NoError(t, err)

		parentState := env.WaitForFlowStatus(t,
			"parent-missing-output", childFlowTimeout,
		)
		assert.Equal(t, api.FlowFailed, parentState.Status)
	})
}

func metaFlowID(meta api.Metadata) api.FlowID {
	switch val := meta[api.MetaParentFlowID].(type) {
	case api.FlowID:
		return val
	case string:
		return api.FlowID(val)
	default:
		return ""
	}
}

func metaStepID(meta api.Metadata) api.StepID {
	switch val := meta[api.MetaParentStepID].(type) {
	case api.StepID:
		return val
	case string:
		return api.StepID(val)
	default:
		return ""
	}
}

func metaToken(meta api.Metadata) api.Token {
	switch val := meta[api.MetaParentWorkItemToken].(type) {
	case api.Token:
		return val
	case string:
		return api.Token(val)
	default:
		return ""
	}
}

func waitForFlowEvents(
	t *testing.T, consumer *timebox.Consumer, timeout time.Duration,
	eventType api.EventType, flowIDs ...api.FlowID,
) {
	t.Helper()

	expected := make(util.Set[api.FlowID], len(flowIDs))
	for _, flowID := range flowIDs {
		expected.Add(flowID)
	}

	filter := helpers.EventDataFilter(
		func(ev *timebox.Event) bool {
			return ev != nil && ev.Type == timebox.EventType(eventType)
		},
		func(data flowEvent) bool {
			if expected.Contains(data.FlowID) {
				expected.Remove(data.FlowID)
				return true
			}
			return false
		},
	)

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for len(expected) > 0 {
		select {
		case ev, ok := <-consumer.Receive():
			if !ok {
				t.Fatalf(
					"event consumer closed before receiving %d events",
					len(flowIDs),
				)
			}
			if ev == nil || !filter(ev) {
				continue
			}
		case <-deadline.C:
			t.Fatalf("timeout waiting for %d events", len(flowIDs))
		}
	}
}
