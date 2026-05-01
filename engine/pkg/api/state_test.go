package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestSetStep(t *testing.T) {
	original := &api.CatalogState{
		Steps: api.Steps{
			"existing": {ID: "existing", Name: "Existing Step"},
		},
	}

	newStep := &api.Step{ID: "new", Name: "New Step"}
	result := original.SetStep("new", newStep)

	assert.Len(t, result.Steps, 2)
	assert.Equal(t, newStep, result.Steps["new"])
	assert.NotNil(t, result.Steps["existing"])
	assert.Len(t, original.Steps, 1)
}

func TestDeleteStep(t *testing.T) {
	original := &api.CatalogState{
		Steps: api.Steps{
			"step1": {ID: "step1"},
			"step2": {ID: "step2"},
		},
	}

	result := original.DeleteStep("step1")

	assert.Len(t, result.Steps, 1)
	assert.Nil(t, result.Steps["step1"])
	assert.NotNil(t, result.Steps["step2"])
	assert.Len(t, original.Steps, 2)
}

func TestSetHealth(t *testing.T) {
	original := &api.NodeState{
		Health: map[api.StepID]api.HealthState{},
	}

	h := api.HealthState{Status: api.HealthHealthy}
	result := original.SetHealth("test-step", h)

	assert.Equal(t, h, result.Health["test-step"])
	assert.Empty(t, original.Health)
}

func TestSetLastSeen(t *testing.T) {
	original := &api.NodeState{LastSeen: time.Unix(1000, 0)}
	newTime := time.Unix(2000, 0)

	result := original.SetLastSeen(newTime)

	assert.True(t, result.LastSeen.Equal(newTime))
	assert.True(t, original.LastSeen.Equal(time.Unix(1000, 0)))
}

func TestSetClusterNode(t *testing.T) {
	node := api.NodeState{}
	original := &api.ClusterState{
		Nodes: map[api.NodeID]api.NodeState{"node-1": node},
	}

	result := original.SetNode("node-2", node)

	assert.Contains(t, result.Nodes, api.NodeID("node-1"))
	assert.Contains(t, result.Nodes, api.NodeID("node-2"))
	assert.NotContains(t, original.Nodes, api.NodeID("node-2"))
}

func TestEnsureClusterNode(t *testing.T) {
	original := &api.ClusterState{
		Nodes: map[api.NodeID]api.NodeState{
			"node-1": {Health: map[api.StepID]api.HealthState{}},
		},
	}

	result := original.EnsureNode("node-2")

	assert.Contains(t, result.Nodes, api.NodeID("node-1"))
	if assert.Contains(t, result.Nodes, api.NodeID("node-2")) {
		assert.Empty(t, result.Nodes["node-2"].Health)
	}
	assert.NotContains(t, original.Nodes, api.NodeID("node-2"))
}

func TestEnsureClusterNodeExisting(t *testing.T) {
	original := &api.ClusterState{
		Nodes: map[api.NodeID]api.NodeState{
			"node-1": {Health: map[api.StepID]api.HealthState{}},
		},
	}

	result := original.EnsureNode("node-1")

	assert.Equal(t, *original, result)
}

func TestSetClusterLastUpdated(t *testing.T) {
	original := &api.ClusterState{}
	newTime := time.Unix(2000, 0)

	result := original.SetLastUpdated(newTime)

	assert.True(t, result.LastUpdated.Equal(newTime))
	assert.True(t, original.LastUpdated.IsZero())
}

func TestSetFlowStatus(t *testing.T) {
	original := &api.FlowState{Status: api.FlowActive}

	result := original.SetStatus(api.FlowCompleted)

	assert.Equal(t, api.FlowCompleted, result.Status)
	assert.Equal(t, api.FlowActive, original.Status)
}

func TestSetAttribute(t *testing.T) {
	original := &api.FlowState{
		Attributes: api.AttributeValues{
			"existing": {{Value: "value"}},
		},
	}

	result := original.SetAttribute("new_attr", &api.AttributeValue{
		Value: "new_value",
		Step:  "test-step",
	})

	assert.Equal(t, "new_value", result.Attributes["new_attr"][0].Value)
	assert.Equal(t,
		api.StepID("test-step"), result.Attributes["new_attr"][0].Step,
	)
	assert.Equal(t, "value", result.Attributes["existing"][0].Value)
	_, ok := original.Attributes["new_attr"]
	assert.False(t, ok)
}

func TestSetExecution(t *testing.T) {
	original := &api.FlowState{
		Executions: api.Executions{
			"existing": {Status: api.StepPending},
		},
	}

	newExec := api.ExecutionState{Status: api.StepActive}
	result := original.SetExecution("new", newExec)

	assert.Len(t, result.Executions, 2)
	assert.Equal(t, newExec, result.Executions["new"])
	assert.Len(t, original.Executions, 1)
}

func TestSetFlowCompleted(t *testing.T) {
	original := &api.FlowState{}
	completedTime := time.Now()

	result := original.SetCompletedAt(completedTime)

	assert.True(t, result.CompletedAt.Equal(completedTime))
	assert.True(t, original.CompletedAt.IsZero())
}

func TestSetFlowError(t *testing.T) {
	original := &api.FlowState{Error: ""}

	result := original.SetError("test error")

	assert.Equal(t, "test error", result.Error)
	assert.Empty(t, original.Error)
}

func TestSetFlowUpdated(t *testing.T) {
	original := &api.FlowState{LastUpdated: time.Unix(1000, 0)}
	newTime := time.Unix(2000, 0)

	result := original.SetLastUpdated(newTime)

	assert.True(t, result.LastUpdated.Equal(newTime))
	assert.True(t, original.LastUpdated.Equal(time.Unix(1000, 0)))
}

func TestSetFlowDeactivated(t *testing.T) {
	original := &api.FlowState{}
	deactivatedAt := time.Now()

	result := original.SetDeactivatedAt(deactivatedAt)

	assert.True(t, result.DeactivatedAt.Equal(deactivatedAt))
	assert.True(t, original.DeactivatedAt.IsZero())
}

func TestGetAttributes(t *testing.T) {
	fl := &api.FlowState{
		Attributes: api.AttributeValues{
			"attr1": {{Value: "value1", Step: "step-1"}},
			"attr2": {{Value: 42, Step: "step-2"}},
			"attr3": {{Value: true, Step: "step-3"}},
		},
	}

	args := fl.GetAttributes()

	assert.Len(t, args, 3)
	assert.Equal(t, "value1", args["attr1"])
	assert.Equal(t, 42, args["attr2"])
	assert.Equal(t, true, args["attr3"])
}

func TestSetExecStatus(t *testing.T) {
	original := &api.ExecutionState{Status: api.StepPending}

	result := original.SetStatus(api.StepActive)

	assert.Equal(t, api.StepActive, result.Status)
	assert.Equal(t, api.StepPending, original.Status)
}

func TestSetStarted(t *testing.T) {
	original := &api.ExecutionState{}
	startTime := time.Now()

	result := original.SetStartedAt(startTime)

	assert.True(t, result.StartedAt.Equal(startTime))
	assert.True(t, original.StartedAt.IsZero())
}

func TestSetExecCompleted(t *testing.T) {
	original := &api.ExecutionState{}
	completedTime := time.Now()

	result := original.SetCompletedAt(completedTime)

	assert.True(t, result.CompletedAt.Equal(completedTime))
	assert.True(t, original.CompletedAt.IsZero())
}

func TestSetInputs(t *testing.T) {
	original := &api.ExecutionState{
		Inputs: api.Args{"existing": "value"},
	}

	newInputs := api.Args{"input1": "value1", "input2": 42}
	result := original.SetInputs(newInputs)

	assert.Len(t, result.Inputs, 2)
	assert.Equal(t, "value1", result.Inputs["input1"])
	assert.Len(t, original.Inputs, 1)
}

func TestSetOutputs(t *testing.T) {
	original := &api.ExecutionState{
		Outputs: api.Args{"existing": "value"},
	}

	newOutputs := api.Args{"output1": "result1", "output2": 100}
	result := original.SetOutputs(newOutputs)

	assert.Len(t, result.Outputs, 2)
	assert.Equal(t, "result1", result.Outputs["output1"])
	assert.Len(t, original.Outputs, 1)
}

func TestSetDuration(t *testing.T) {
	original := &api.ExecutionState{Duration: 100}

	result := original.SetDuration(500)

	assert.EqualValues(t, 500, result.Duration)
	assert.EqualValues(t, 100, original.Duration)
}

func TestSetExecError(t *testing.T) {
	original := &api.ExecutionState{Error: ""}

	result := original.SetError("execution error")

	assert.Equal(t, "execution error", result.Error)
	assert.Empty(t, original.Error)
}

func TestSetWorkItem(t *testing.T) {
	original := &api.ExecutionState{
		WorkItems: map[api.Token]api.WorkState{},
	}

	work := api.WorkState{
		Status: api.WorkPending,
	}

	result := original.SetWorkItem("work-1", work)

	assert.Len(t, result.WorkItems, 1)
	assert.Equal(t, work, result.WorkItems["work-1"])
	assert.Empty(t, original.WorkItems)
}

func TestSetHealthStatus(t *testing.T) {
	original := &api.HealthState{Status: api.HealthHealthy}

	result := original.SetStatus(api.HealthUnhealthy)

	assert.Equal(t, api.HealthUnhealthy, result.Status)
	assert.Equal(t, api.HealthHealthy, original.Status)
}

func TestSetHealthError(t *testing.T) {
	original := &api.HealthState{Error: ""}

	result := original.SetError("health check failed")

	assert.Equal(t, "health check failed", result.Error)
	assert.Empty(t, original.Error)
}

func TestSetWorkStatus(t *testing.T) {
	original := &api.WorkState{
		Status: api.WorkPending,
	}

	result := original.SetStatus(api.WorkActive)

	assert.Equal(t, api.WorkActive, result.Status)
	assert.Equal(t, api.WorkPending, original.Status)
}

func TestSetWorkStarted(t *testing.T) {
	original := &api.WorkState{}
	startTime := time.Now()

	result := original.SetStartedAt(startTime)

	assert.True(t, result.StartedAt.Equal(startTime))
	assert.True(t, original.StartedAt.IsZero())
}

func TestSetWorkCompleted(t *testing.T) {
	original := &api.WorkState{}
	completedTime := time.Now()

	result := original.SetCompletedAt(completedTime)

	assert.True(t, result.CompletedAt.Equal(completedTime))
	assert.True(t, original.CompletedAt.IsZero())
}

func TestSetRetryCount(t *testing.T) {
	original := &api.WorkState{
		RetryCount: 0,
	}

	result := original.SetRetryCount(3)

	assert.Equal(t, 3, result.RetryCount)
	assert.Equal(t, 0, original.RetryCount)
}

func TestSetNextRetry(t *testing.T) {
	original := &api.WorkState{}
	nextRetry := time.Now().Add(time.Minute)

	result := original.SetNextRetryAt(nextRetry)

	assert.True(t, result.NextRetryAt.Equal(nextRetry))
	assert.True(t, original.NextRetryAt.IsZero())
}

func TestSetWorkError(t *testing.T) {
	original := &api.WorkState{
		Error: "",
	}

	result := original.SetError("work item failed")

	assert.Equal(t, "work item failed", result.Error)
	assert.Empty(t, original.Error)
}

func TestSetWorkOutputs(t *testing.T) {
	original := &api.WorkState{
		Outputs: api.Args{},
	}

	outputs := api.Args{"result": "success", "count": 42}
	result := original.SetOutputs(outputs)

	assert.Len(t, result.Outputs, 2)
	assert.Equal(t, "success", result.Outputs["result"])
	assert.Equal(t, 42, result.Outputs["count"])
	assert.Empty(t, original.Outputs)
}

func TestFlowChain(t *testing.T) {
	original := &api.FlowState{
		ID:         "test-flow",
		Status:     api.FlowActive,
		Attributes: api.AttributeValues{},
		Executions: api.Executions{},
	}

	result := original.
		SetStatus(api.FlowCompleted).
		SetAttribute(
			"attr1", &api.AttributeValue{Value: "value1", Step: "step1"},
		).
		SetAttribute("attr2", &api.AttributeValue{Value: 42, Step: "step2"})

	assert.Equal(t, api.FlowCompleted, result.Status)
	assert.Equal(t, "value1", result.Attributes["attr1"][0].Value)
	assert.Equal(t, api.StepID("step1"), result.Attributes["attr1"][0].Step)
	assert.Equal(t, 42, result.Attributes["attr2"][0].Value)
	assert.Equal(t, api.StepID("step2"), result.Attributes["attr2"][0].Step)
	assert.Equal(t, api.FlowActive, original.Status)
}

func TestExecChain(t *testing.T) {
	original := &api.ExecutionState{Status: api.StepPending}

	startTime := time.Now()
	completedTime := startTime.Add(time.Second)

	result := original.
		SetStatus(api.StepActive).
		SetStartedAt(startTime).
		SetInputs(api.Args{"input": "value"}).
		SetStatus(api.StepCompleted).
		SetOutputs(api.Args{"output": "result"}).
		SetCompletedAt(completedTime).
		SetDuration(1000)

	assert.Equal(t, api.StepCompleted, result.Status)
	assert.True(t, result.StartedAt.Equal(startTime))
	assert.EqualValues(t, 1000, result.Duration)
	assert.Equal(t, api.StepPending, original.Status)
}

func TestWorkChain(t *testing.T) {
	original := &api.WorkState{
		Status: api.WorkPending,
	}

	startTime := time.Now()
	completedTime := startTime.Add(time.Second)
	outputs := api.Args{"result": "success"}

	result := original.
		SetStatus(api.WorkActive).
		SetStartedAt(startTime).
		SetStatus(api.WorkSucceeded).
		SetCompletedAt(completedTime).
		SetOutputs(outputs)

	assert.Equal(t, api.WorkSucceeded, result.Status)
	assert.True(t, result.StartedAt.Equal(startTime))
	assert.True(t, result.CompletedAt.Equal(completedTime))
	assert.Equal(t, outputs, result.Outputs)
	assert.Equal(t, api.WorkPending, original.Status)
}
