package api

import (
	"time"

	"github.com/kode4food/timebox"
)

type (
	// StepRegisteredEvent is emitted when a step is registered with the engine
	StepRegisteredEvent struct {
		Step *Step `json:"step"`
	}

	// StepUnregisteredEvent is emitted when a step is removed from the engine
	StepUnregisteredEvent struct {
		StepID timebox.ID `json:"step_id"`
	}

	// StepHealthChangedEvent is emitted when a step's health status changes
	StepHealthChangedEvent struct {
		StepID timebox.ID   `json:"step_id"`
		Status HealthStatus `json:"status"`
		Error  string       `json:"error,omitempty"`
	}

	// FlowStartedEvent is emitted when a flow execution begins
	FlowStartedEvent struct {
		Plan     *ExecutionPlan `json:"plan"`
		Init     Args           `json:"init"`
		Metadata Metadata       `json:"metadata,omitempty"`
		FlowID   timebox.ID     `json:"flow_id"`
	}

	// FlowCompletedEvent is emitted when a flow completes successfully
	FlowCompletedEvent struct {
		Result Args       `json:"result"`
		FlowID timebox.ID `json:"flow_id"`
	}

	// FlowFailedEvent is emitted when a flow fails
	FlowFailedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
		Error  string     `json:"error"`
	}

	// StepStartedEvent is emitted when a step begins execution
	StepStartedEvent struct {
		Inputs Args       `json:"inputs"`
		FlowID timebox.ID `json:"flow_id"`
		StepID timebox.ID `json:"step_id"`
	}

	// StepCompletedEvent is emitted when a step completes successfully
	StepCompletedEvent struct {
		Outputs  Args       `json:"outputs"`
		FlowID   timebox.ID `json:"flow_id"`
		StepID   timebox.ID `json:"step_id"`
		Duration int64      `json:"duration"`
	}

	// StepFailedEvent is emitted when a step fails
	StepFailedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
		StepID timebox.ID `json:"step_id"`
		Error  string     `json:"error"`
	}

	// StepSkippedEvent is emitted when a step is skipped due to predicate
	StepSkippedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
		StepID timebox.ID `json:"step_id"`
		Reason string     `json:"reason"`
	}

	// AttributeSetEvent is emitted when a flow attribute value is set
	AttributeSetEvent struct {
		Value  any        `json:"value"`
		FlowID timebox.ID `json:"flow_id"`
		StepID timebox.ID `json:"step_id"`
		Key    Name       `json:"key"`
	}

	// WorkStartedEvent is emitted when a work item begins execution
	WorkStartedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
		StepID timebox.ID `json:"step_id"`
		Token  Token      `json:"token"`
		Inputs Args       `json:"inputs"`
	}

	// WorkCompletedEvent is emitted when a work item completes successfully
	WorkCompletedEvent struct {
		FlowID  timebox.ID `json:"flow_id"`
		StepID  timebox.ID `json:"step_id"`
		Token   Token      `json:"token"`
		Outputs Args       `json:"outputs"`
	}

	// WorkFailedEvent is emitted when a work item fails
	WorkFailedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
		StepID timebox.ID `json:"step_id"`
		Token  Token      `json:"token"`
		Error  string     `json:"error"`
	}

	// RetryScheduledEvent is emitted when a failed work item is to be retried
	RetryScheduledEvent struct {
		FlowID      timebox.ID `json:"flow_id"`
		StepID      timebox.ID `json:"step_id"`
		Token       Token      `json:"token"`
		RetryCount  int        `json:"retry_count"`
		NextRetryAt time.Time  `json:"next_retry_at"`
		Error       string     `json:"error"`
	}

	// FlowActivatedEvent is emitted when a flow becomes active
	FlowActivatedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
	}

	// FlowDeactivatedEvent is emitted when a flow becomes inactive
	FlowDeactivatedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
	}
)

const (
	EventTypeStepRegistered    timebox.EventType = "step_registered"
	EventTypeStepUnregistered  timebox.EventType = "step_unregistered"
	EventTypeStepHealthChanged timebox.EventType = "step_health_changed"
	EventTypeFlowActivated     timebox.EventType = "flow_activated"
	EventTypeFlowDeactivated   timebox.EventType = "flow_deactivated"
	EventTypeFlowStarted       timebox.EventType = "flow_started"
	EventTypeFlowCompleted     timebox.EventType = "flow_completed"
	EventTypeFlowFailed        timebox.EventType = "flow_failed"
	EventTypeAttributeSet      timebox.EventType = "attribute_set"
	EventTypeStepStarted       timebox.EventType = "step_started"
	EventTypeStepCompleted     timebox.EventType = "step_completed"
	EventTypeStepFailed        timebox.EventType = "step_failed"
	EventTypeStepSkipped       timebox.EventType = "step_skipped"
	EventTypeWorkStarted       timebox.EventType = "work_started"
	EventTypeWorkCompleted     timebox.EventType = "work_completed"
	EventTypeWorkFailed        timebox.EventType = "work_failed"
	EventTypeRetryScheduled    timebox.EventType = "retry_scheduled"
)
