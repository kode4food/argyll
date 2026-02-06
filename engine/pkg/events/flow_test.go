package events_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func TestNewFlowState(t *testing.T) {
	state := events.NewFlowState()

	assert.NotNil(t, state)
	assert.NotNil(t, state.Attributes)
	assert.NotNil(t, state.Executions)
	assert.Empty(t, state.Attributes)
	assert.Empty(t, state.Executions)
}

func TestIsFlowEvent(t *testing.T) {
	flowEvent := &timebox.Event{
		AggregateID: events.FlowKey("test-flow"),
	}
	engineEvent := &timebox.Event{
		AggregateID: events.EngineKey,
	}

	assert.True(t, events.IsFlowEvent(flowEvent))
	assert.False(t, events.IsFlowEvent(engineEvent))
}

func TestFlowStarted(t *testing.T) {
	initialState := events.NewFlowState()
	now := time.Now()

	plan := &api.ExecutionPlan{
		Steps: api.Steps{
			"step1": {ID: "step1", Name: "Step 1"},
			"step2": {ID: "step2", Name: "Step 2"},
		},
	}

	eventData := api.FlowStartedEvent{
		FlowID: "test-flow",
		Plan:   plan,
		Init: api.Args{
			"input1": "value1",
			"input2": 42,
		},
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Equal(t, api.FlowID("test-flow"), result.ID)
	assert.Equal(t, api.FlowActive, result.Status)
	assert.Equal(t, plan, result.Plan)
	assert.Len(t, result.Attributes, 2)
	assert.Equal(t, "value1", result.Attributes["input1"].Value)
	assert.Equal(t, float64(42), result.Attributes["input2"].Value)
	assert.Len(t, result.Executions, 2)
	assert.True(t, result.CreatedAt.Equal(now))
}

func TestFlowCompleted(t *testing.T) {
	initialState := &api.FlowState{
		ID:     "test-flow",
		Status: api.FlowActive,
	}
	now := time.Now()

	eventData := api.FlowCompletedEvent{FlowID: "test-flow"}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeFlowCompleted),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	assert.Equal(t, api.FlowCompleted, result.Status)
	assert.True(t, result.CompletedAt.Equal(now))
}

func TestFlowFailed(t *testing.T) {
	initialState := &api.FlowState{
		ID:     "test-flow",
		Status: api.FlowActive,
	}
	now := time.Now()

	eventData := api.FlowFailedEvent{
		FlowID: "test-flow",
		Error:  "execution failed",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeFlowFailed),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	assert.Equal(t, api.FlowFailed, result.Status)
	assert.Equal(t, "execution failed", result.Error)
	assert.True(t, result.CompletedAt.Equal(now))
}

func TestStepStarted(t *testing.T) {
	initialState := &api.FlowState{
		ID:     "test-flow",
		Status: api.FlowActive,
		Executions: api.Executions{
			"step1": {Status: api.StepPending},
		},
	}
	now := time.Now()

	eventData := api.StepStartedEvent{
		StepID: "step1",
		Inputs: api.Args{"input": "value"},
		WorkItems: map[api.Token]api.Args{
			"token1": {"work_input": "work_value"},
		},
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeStepStarted),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	exec := result.Executions["step1"]
	assert.Equal(t, api.StepActive, exec.Status)
	assert.True(t, exec.StartedAt.Equal(now))
	assert.Equal(t, "value", exec.Inputs["input"])
	assert.Len(t, exec.WorkItems, 1)
}

func TestStepCompleted(t *testing.T) {
	initialState := &api.FlowState{
		ID:     "test-flow",
		Status: api.FlowActive,
		Executions: api.Executions{
			"step1": {Status: api.StepActive},
		},
	}
	now := time.Now()

	eventData := api.StepCompletedEvent{
		StepID:   "step1",
		Duration: 1000,
		Outputs:  api.Args{"result": "success"},
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeStepCompleted),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	exec := result.Executions["step1"]
	assert.Equal(t, api.StepCompleted, exec.Status)
	assert.True(t, exec.CompletedAt.Equal(now))
	assert.EqualValues(t, 1000, exec.Duration)
	assert.Equal(t, "success", exec.Outputs["result"])
}

func TestStepFailed(t *testing.T) {
	initialState := &api.FlowState{
		ID:     "test-flow",
		Status: api.FlowActive,
		Executions: api.Executions{
			"step1": {Status: api.StepActive},
		},
	}
	now := time.Now()

	eventData := api.StepFailedEvent{
		StepID: "step1",
		Error:  "step execution failed",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeStepFailed),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	exec := result.Executions["step1"]
	assert.Equal(t, api.StepFailed, exec.Status)
	assert.Equal(t, "step execution failed", exec.Error)
}

func TestStepSkipped(t *testing.T) {
	initialState := &api.FlowState{
		ID:     "test-flow",
		Status: api.FlowActive,
		Executions: api.Executions{
			"step1": {Status: api.StepPending},
		},
	}
	now := time.Now()

	eventData := api.StepSkippedEvent{
		StepID: "step1",
		Reason: "predicate evaluated to false",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeStepSkipped),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	exec := result.Executions["step1"]
	assert.Equal(t, api.StepSkipped, exec.Status)
	assert.Equal(t, "predicate evaluated to false", exec.Error)
}

func TestAttributeSet(t *testing.T) {
	initialState := &api.FlowState{
		ID:         "test-flow",
		Status:     api.FlowActive,
		Attributes: api.AttributeValues{},
	}
	now := time.Now()

	eventData := api.AttributeSetEvent{
		StepID: "step1",
		Key:    "result",
		Value:  "test-value",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeAttributeSet),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	assert.Len(t, result.Attributes, 1)
	assert.NotNil(t, result.Attributes["result"])
	assert.Equal(t, "test-value", result.Attributes["result"].Value)
	assert.Equal(t, api.StepID("step1"), result.Attributes["result"].Step)
}

func TestWorkStarted(t *testing.T) {
	initialState := &api.FlowState{
		ID:     "test-flow",
		Status: api.FlowActive,
		Executions: api.Executions{
			"step1": {
				Status: api.StepActive,
				WorkItems: api.WorkItems{
					"token1": {
						Status: api.WorkPending,
						Inputs: api.Args{"input": "value"},
					},
				},
			},
		},
	}
	now := time.Now()

	eventData := api.WorkStartedEvent{
		StepID: "step1",
		Token:  "token1",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeWorkStarted),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	exec := result.Executions["step1"]
	workItem := exec.WorkItems["token1"]
	assert.Equal(t, api.WorkActive, workItem.Status)
	assert.True(t, workItem.StartedAt.Equal(now))
}

func TestWorkSucceeded(t *testing.T) {
	initialState := &api.FlowState{
		ID:     "test-flow",
		Status: api.FlowActive,
		Executions: api.Executions{
			"step1": {
				Status: api.StepActive,
				WorkItems: api.WorkItems{
					"token1": {Status: api.WorkActive},
				},
			},
		},
	}
	now := time.Now()

	eventData := api.WorkSucceededEvent{
		StepID:  "step1",
		Token:   "token1",
		Outputs: api.Args{"work_result": "success"},
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeWorkSucceeded),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	exec := result.Executions["step1"]
	workItem := exec.WorkItems["token1"]
	assert.Equal(t, api.WorkSucceeded, workItem.Status)
	assert.True(t, workItem.CompletedAt.Equal(now))
	assert.Equal(t, "success", workItem.Outputs["work_result"])
}

func TestWorkFailed(t *testing.T) {
	initialState := &api.FlowState{
		ID:     "test-flow",
		Status: api.FlowActive,
		Executions: api.Executions{
			"step1": {
				Status: api.StepActive,
				WorkItems: api.WorkItems{
					"token1": {Status: api.WorkActive},
				},
			},
		},
	}
	now := time.Now()

	eventData := api.WorkFailedEvent{
		StepID: "step1",
		Token:  "token1",
		Error:  "work execution failed",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeWorkFailed),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	exec := result.Executions["step1"]
	workItem := exec.WorkItems["token1"]
	assert.Equal(t, api.WorkFailed, workItem.Status)
	assert.Equal(t, "work execution failed", workItem.Error)
}

func TestWorkNotCompleted(t *testing.T) {
	initialState := &api.FlowState{
		ID:     "test-flow",
		Status: api.FlowActive,
		Executions: api.Executions{
			"step1": {
				Status: api.StepActive,
				WorkItems: api.WorkItems{
					"token1": {Status: api.WorkActive},
				},
			},
		},
	}
	now := time.Now()

	eventData := api.WorkNotCompletedEvent{
		StepID: "step1",
		Token:  "token1",
		Error:  "work not completed",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeWorkNotCompleted),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	exec := result.Executions["step1"]
	workItem := exec.WorkItems["token1"]
	assert.Equal(t, api.WorkNotCompleted, workItem.Status)
	assert.Equal(t, "work not completed", workItem.Error)
}

func TestRetryScheduled(t *testing.T) {
	initialState := &api.FlowState{
		ID:     "test-flow",
		Status: api.FlowActive,
		Executions: api.Executions{
			"step1": {
				Status: api.StepActive,
				WorkItems: api.WorkItems{
					"token1": {
						Status:     api.WorkFailed,
						Error:      "previous error",
						RetryCount: 0,
					},
				},
			},
		},
	}
	now := time.Now()
	nextRetry := now.Add(5 * time.Second)

	eventData := api.RetryScheduledEvent{
		StepID:      "step1",
		Token:       "token1",
		RetryCount:  1,
		NextRetryAt: nextRetry,
		Error:       "previous error",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeRetryScheduled),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]
	result := applier(initialState, event)

	exec := result.Executions["step1"]
	workItem := exec.WorkItems["token1"]
	assert.Equal(t, api.WorkPending, workItem.Status)
	assert.Equal(t, 1, workItem.RetryCount)
	assert.True(t, workItem.NextRetryAt.Equal(nextRetry))
}

func TestMissingExecution(t *testing.T) {
	initialState := &api.FlowState{
		ID:         "test-flow",
		Status:     api.FlowActive,
		Executions: api.Executions{},
	}
	now := time.Now()

	eventData := api.StepCompletedEvent{
		StepID:   "nonexistent",
		Duration: 1000,
		Outputs:  api.Args{},
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.FlowKey("test-flow"),
		Type:        timebox.EventType(api.EventTypeStepCompleted),
		Data:        data,
	}

	applier := events.FlowAppliers[event.Type]

	assert.Panics(t, func() {
		applier(initialState, event)
	})
}
