package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestEngineSetStep(t *testing.T) {
	original := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
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

func TestEngineDeleteStep(t *testing.T) {
	original := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
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

func TestEngineSetHealth(t *testing.T) {
	original := &api.EngineState{
		Health: map[api.StepID]*api.HealthState{},
	}

	health := &api.HealthState{Status: api.HealthHealthy}
	result := original.SetHealth("test-step", health)

	assert.Equal(t, health, result.Health["test-step"])
	assert.Empty(t, original.Health)
}

func TestEngineSetUpdated(t *testing.T) {
	original := &api.EngineState{LastUpdated: time.Unix(1000, 0)}
	newTime := time.Unix(2000, 0)

	result := original.SetLastUpdated(newTime)

	assert.True(t, result.LastUpdated.Equal(newTime))
	assert.True(t, original.LastUpdated.Equal(time.Unix(1000, 0)))
}

func TestFlowSetStatus(t *testing.T) {
	original := &api.FlowState{Status: api.FlowPending}

	result := original.SetStatus(api.FlowActive)

	assert.Equal(t, api.FlowActive, result.Status)
	assert.Equal(t, api.FlowPending, original.Status)
}

func TestFlowSetAttribute(t *testing.T) {
	original := &api.FlowState{
		Attributes: map[api.Name]*api.AttributeValue{
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

func TestFlowSetExecution(t *testing.T) {
	original := &api.FlowState{
		Executions: map[api.StepID]*api.ExecutionState{
			"existing": {Status: api.StepPending},
		},
	}

	newExec := &api.ExecutionState{Status: api.StepActive}
	result := original.SetExecution("new", newExec)

	assert.Len(t, result.Executions, 2)
	assert.Equal(t, newExec, result.Executions["new"])
	assert.Len(t, original.Executions, 1)
}

func TestFlowSetCompleted(t *testing.T) {
	original := &api.FlowState{}
	completedTime := time.Now()

	result := original.SetCompletedAt(completedTime)

	assert.True(t, result.CompletedAt.Equal(completedTime))
	assert.True(t, original.CompletedAt.IsZero())
}

func TestFlowSetError(t *testing.T) {
	original := &api.FlowState{Error: ""}

	result := original.SetError("test error")

	assert.Equal(t, "test error", result.Error)
	assert.Empty(t, original.Error)
}

func TestFlowSetUpdated(t *testing.T) {
	original := &api.FlowState{LastUpdated: time.Unix(1000, 0)}
	newTime := time.Unix(2000, 0)

	result := original.SetLastUpdated(newTime)

	assert.True(t, result.LastUpdated.Equal(newTime))
	assert.True(t, original.LastUpdated.Equal(time.Unix(1000, 0)))
}

func TestExecutionSetStatus(t *testing.T) {
	original := &api.ExecutionState{Status: api.StepPending}

	result := original.SetStatus(api.StepActive)

	assert.Equal(t, api.StepActive, result.Status)
	assert.Equal(t, api.StepPending, original.Status)
}

func TestExecutionSetStarted(t *testing.T) {
	original := &api.ExecutionState{}
	startTime := time.Now()

	result := original.SetStartedAt(startTime)

	assert.True(t, result.StartedAt.Equal(startTime))
	assert.True(t, original.StartedAt.IsZero())
}

func TestExecutionSetCompleted(t *testing.T) {
	original := &api.ExecutionState{}
	completedTime := time.Now()

	result := original.SetCompletedAt(completedTime)

	assert.True(t, result.CompletedAt.Equal(completedTime))
	assert.True(t, original.CompletedAt.IsZero())
}

func TestExecutionSetInputs(t *testing.T) {
	original := &api.ExecutionState{
		Inputs: api.Args{"existing": "value"},
	}

	newInputs := api.Args{"input1": "value1", "input2": 42}
	result := original.SetInputs(newInputs)

	assert.Len(t, result.Inputs, 2)
	assert.Equal(t, "value1", result.Inputs["input1"])
	assert.Len(t, original.Inputs, 1)
}

func TestExecutionSetOutputs(t *testing.T) {
	original := &api.ExecutionState{
		Outputs: api.Args{"existing": "value"},
	}

	newOutputs := api.Args{"output1": "result1", "output2": 100}
	result := original.SetOutputs(newOutputs)

	assert.Len(t, result.Outputs, 2)
	assert.Equal(t, "result1", result.Outputs["output1"])
	assert.Len(t, original.Outputs, 1)
}

func TestExecutionSetDuration(t *testing.T) {
	original := &api.ExecutionState{Duration: 100}

	result := original.SetDuration(500)

	assert.EqualValues(t, 500, result.Duration)
	assert.EqualValues(t, 100, original.Duration)
}

func TestExecutionSetError(t *testing.T) {
	original := &api.ExecutionState{Error: ""}

	result := original.SetError("execution error")

	assert.Equal(t, "execution error", result.Error)
	assert.Empty(t, original.Error)
}

func TestHealthSetStatus(t *testing.T) {
	original := &api.HealthState{Status: api.HealthHealthy}

	result := original.SetStatus(api.HealthUnhealthy)

	assert.Equal(t, api.HealthUnhealthy, result.Status)
	assert.Equal(t, api.HealthHealthy, original.Status)
}

func TestHealthSetError(t *testing.T) {
	original := &api.HealthState{Error: ""}

	result := original.SetError("health check failed")

	assert.Equal(t, "health check failed", result.Error)
	assert.Empty(t, original.Error)
}

func TestFlowChaining(t *testing.T) {
	original := &api.FlowState{
		ID:         "test-flow",
		Status:     api.FlowPending,
		Attributes: map[api.Name]*api.AttributeValue{},
		Executions: map[api.StepID]*api.ExecutionState{},
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

func TestExecutionChaining(t *testing.T) {
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
