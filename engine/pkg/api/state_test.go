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
	original := &api.PartitionState{
		Health: map[api.StepID]*api.HealthState{},
	}

	health := &api.HealthState{Status: api.HealthHealthy}
	result := original.SetHealth("test-step", health)

	assert.Equal(t, health, result.Health["test-step"])
	assert.Empty(t, original.Health)
}

func TestSetPartitionUpdated(t *testing.T) {
	original := &api.PartitionState{LastUpdated: time.Unix(1000, 0)}
	newTime := time.Unix(2000, 0)

	result := original.SetLastUpdated(newTime)

	assert.True(t, result.LastUpdated.Equal(newTime))
	assert.True(t, original.LastUpdated.Equal(time.Unix(1000, 0)))
}

func TestSetActiveFlow(t *testing.T) {
	original := &api.PartitionState{
		Active: map[api.FlowID]*api.ActiveFlow{},
	}

	flowInfo := &api.ActiveFlow{
		StartedAt:  time.Now(),
		LastActive: time.Now(),
	}

	result := original.SetActiveFlow("flow-1", flowInfo)

	assert.Len(t, result.Active, 1)
	assert.Equal(t, flowInfo, result.Active["flow-1"])
	assert.Empty(t, original.Active)
}

func TestDeleteActiveFlow(t *testing.T) {
	original := &api.PartitionState{
		Active: map[api.FlowID]*api.ActiveFlow{
			"flow-1": {},
			"flow-2": {},
		},
	}

	result := original.DeleteActiveFlow("flow-1")

	assert.Len(t, result.Active, 1)
	assert.Nil(t, result.Active["flow-1"])
	assert.NotNil(t, result.Active["flow-2"])
	assert.Len(t, original.Active, 2)
}

func TestSetFlowDigest(t *testing.T) {
	now := time.Now()
	original := &api.PartitionState{
		FlowDigests: map[api.FlowID]*api.FlowDigest{},
	}

	digest := &api.FlowDigest{
		Status:    api.FlowActive,
		CreatedAt: now,
	}
	result := original.SetFlowDigest("flow-1", digest)

	assert.Len(t, result.FlowDigests, 1)
	assert.Equal(t, digest, result.FlowDigests["flow-1"])
	assert.Empty(t, original.FlowDigests)
}

func TestDeleteFlowDigest(t *testing.T) {
	original := &api.PartitionState{
		FlowDigests: map[api.FlowID]*api.FlowDigest{
			"flow-1": {Status: api.FlowActive},
			"flow-2": {Status: api.FlowActive},
		},
	}

	result := original.DeleteFlowDigest("flow-1")

	assert.Len(t, result.FlowDigests, 1)
	assert.Nil(t, result.FlowDigests["flow-1"])
	assert.NotNil(t, result.FlowDigests["flow-2"])
	assert.Len(t, original.FlowDigests, 2)
}

func TestAddDeactivated(t *testing.T) {
	now := time.Now()
	original := &api.PartitionState{
		Deactivated: []*api.DeactivatedFlow{
			{FlowID: "flow-1", DeactivatedAt: now.Add(-time.Hour)},
		},
	}

	result := original.AddDeactivated(&api.DeactivatedFlow{
		FlowID:        "flow-2",
		DeactivatedAt: now,
	})

	assert.Len(t, result.Deactivated, 2)
	assert.Equal(t, api.FlowID("flow-1"), result.Deactivated[0].FlowID)
	assert.Equal(t, api.FlowID("flow-2"), result.Deactivated[1].FlowID)
	assert.Len(t, original.Deactivated, 1)
}

func TestRemoveDeactivated(t *testing.T) {
	now := time.Now()
	original := &api.PartitionState{
		Deactivated: []*api.DeactivatedFlow{
			{FlowID: "flow-1", DeactivatedAt: now.Add(-2 * time.Hour)},
			{FlowID: "flow-2", DeactivatedAt: now.Add(-time.Hour)},
			{FlowID: "flow-3", DeactivatedAt: now},
		},
	}

	result := original.RemoveDeactivated("flow-2")

	assert.Len(t, result.Deactivated, 2)
	assert.Equal(t, api.FlowID("flow-1"), result.Deactivated[0].FlowID)
	assert.Equal(t, api.FlowID("flow-3"), result.Deactivated[1].FlowID)
	assert.Len(t, original.Deactivated, 3)
}

func TestRemoveDeactivatedNotFound(t *testing.T) {
	now := time.Now()
	original := &api.PartitionState{
		Deactivated: []*api.DeactivatedFlow{
			{FlowID: "flow-1", DeactivatedAt: now},
		},
	}

	result := original.RemoveDeactivated("flow-missing")

	assert.Same(t, original, result)
}

func TestAddArchiving(t *testing.T) {
	now := time.Now()
	original := &api.PartitionState{
		Archiving: map[api.FlowID]time.Time{
			"flow-1": now.Add(-time.Hour),
		},
	}

	result := original.AddArchiving("flow-2", now)

	assert.Len(t, result.Archiving, 2)
	assert.True(t, result.Archiving["flow-1"].Equal(now.Add(-time.Hour)))
	assert.True(t, result.Archiving["flow-2"].Equal(now))
	assert.Len(t, original.Archiving, 1)
}

func TestAddArchivingReplaces(t *testing.T) {
	now := time.Now()
	original := &api.PartitionState{
		Archiving: map[api.FlowID]time.Time{
			"flow-1": now.Add(-time.Hour),
		},
	}

	result := original.AddArchiving("flow-1", now)

	assert.Len(t, result.Archiving, 1)
	assert.True(t, result.Archiving["flow-1"].Equal(now))
	assert.True(t, original.Archiving["flow-1"].Equal(now.Add(-time.Hour)))
	assert.Len(t, original.Archiving, 1)
}

func TestRemoveArchiving(t *testing.T) {
	now := time.Now()
	original := &api.PartitionState{
		Archiving: map[api.FlowID]time.Time{
			"flow-1": now.Add(-2 * time.Hour),
			"flow-2": now.Add(-time.Hour),
			"flow-3": now,
		},
	}

	result := original.RemoveArchiving("flow-2")

	assert.Len(t, result.Archiving, 2)
	assert.True(t, result.Archiving["flow-1"].Equal(now.Add(-2*time.Hour)))
	_, ok := result.Archiving["flow-2"]
	assert.False(t, ok)
	assert.True(t, result.Archiving["flow-3"].Equal(now))
	assert.Len(t, original.Archiving, 3)
}

func TestRemoveArchivingNotFound(t *testing.T) {
	now := time.Now()
	original := &api.PartitionState{
		Archiving: map[api.FlowID]time.Time{
			"flow-1": now,
		},
	}

	result := original.RemoveArchiving("flow-missing")

	assert.Same(t, original, result)
}

func TestSetFlowStatus(t *testing.T) {
	original := &api.FlowState{Status: api.FlowPending}

	result := original.SetStatus(api.FlowActive)

	assert.Equal(t, api.FlowActive, result.Status)
	assert.Equal(t, api.FlowPending, original.Status)
}

func TestSetAttribute(t *testing.T) {
	original := &api.FlowState{
		Attributes: api.AttributeValues{
			"existing": {Value: "value"},
		},
	}

	result := original.SetAttribute("new_attr", &api.AttributeValue{
		Value: "new_value",
		Step:  "test-step",
	})

	assert.Equal(t, "new_value", result.Attributes["new_attr"].Value)
	assert.Equal(t, api.StepID("test-step"), result.Attributes["new_attr"].Step)
	assert.Equal(t, "value", result.Attributes["existing"].Value)
	_, ok := original.Attributes["new_attr"]
	assert.False(t, ok)
}

func TestSetExecution(t *testing.T) {
	original := &api.FlowState{
		Executions: api.Executions{
			"existing": {Status: api.StepPending},
		},
	}

	newExec := &api.ExecutionState{Status: api.StepActive}
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

func TestGetAttributes(t *testing.T) {
	flow := &api.FlowState{
		Attributes: api.AttributeValues{
			"attr1": {Value: "value1", Step: "step-1"},
			"attr2": {Value: 42, Step: "step-2"},
			"attr3": {Value: true, Step: "step-3"},
		},
	}

	args := flow.GetAttributes()

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
		WorkItems: map[api.Token]*api.WorkState{},
	}

	workItem := &api.WorkState{
		Status: api.WorkPending,
	}

	result := original.SetWorkItem("work-1", workItem)

	assert.Len(t, result.WorkItems, 1)
	assert.Equal(t, workItem, result.WorkItems["work-1"])
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
		Status:     api.FlowPending,
		Attributes: api.AttributeValues{},
		Executions: api.Executions{},
	}

	result := original.
		SetStatus(api.FlowActive).
		SetAttribute(
			"attr1", &api.AttributeValue{Value: "value1", Step: "step1"},
		).
		SetAttribute("attr2", &api.AttributeValue{Value: 42, Step: "step2"})

	assert.Equal(t, api.FlowActive, result.Status)
	assert.Equal(t, "value1", result.Attributes["attr1"].Value)
	assert.Equal(t, api.StepID("step1"), result.Attributes["attr1"].Step)
	assert.Equal(t, 42, result.Attributes["attr2"].Value)
	assert.Equal(t, api.StepID("step2"), result.Attributes["attr2"].Step)
	assert.Equal(t, api.FlowPending, original.Status)
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
