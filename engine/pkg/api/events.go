package api

import (
	"time"

	"github.com/kode4food/timebox"
)

type (
	StepRegisteredEvent struct {
		Step *Step `json:"step"`
	}

	StepUnregisteredEvent struct {
		StepID timebox.ID `json:"step_id"`
	}

	StepHealthChangedEvent struct {
		StepID timebox.ID   `json:"step_id"`
		Status HealthStatus `json:"status"`
		Error  string       `json:"error,omitempty"`
	}

	WorkflowStartedEvent struct {
		Plan     *ExecutionPlan `json:"plan"`
		Init     Args           `json:"init"`
		Metadata Metadata       `json:"metadata,omitempty"`
		FlowID   timebox.ID     `json:"flow_id"`
	}

	WorkflowCompletedEvent struct {
		Result Args       `json:"result"`
		FlowID timebox.ID `json:"flow_id"`
	}

	WorkflowFailedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
		Error  string     `json:"error"`
	}

	StepStartedEvent struct {
		Inputs Args       `json:"inputs"`
		FlowID timebox.ID `json:"flow_id"`
		StepID timebox.ID `json:"step_id"`
	}

	StepCompletedEvent struct {
		Outputs  Args       `json:"outputs"`
		FlowID   timebox.ID `json:"flow_id"`
		StepID   timebox.ID `json:"step_id"`
		Duration int64      `json:"duration"`
	}

	StepFailedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
		StepID timebox.ID `json:"step_id"`
		Error  string     `json:"error"`
	}

	StepSkippedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
		StepID timebox.ID `json:"step_id"`
		Reason string     `json:"reason"`
	}

	AttributeSetEvent struct {
		Value  any        `json:"value"`
		FlowID timebox.ID `json:"flow_id"`
		StepID timebox.ID `json:"step_id"`
		Key    Name       `json:"key"`
	}

	WorkStartedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
		StepID timebox.ID `json:"step_id"`
		Token  Token      `json:"token"`
		Inputs Args       `json:"inputs"`
	}

	WorkCompletedEvent struct {
		FlowID  timebox.ID `json:"flow_id"`
		StepID  timebox.ID `json:"step_id"`
		Token   Token      `json:"token"`
		Outputs Args       `json:"outputs"`
	}

	WorkFailedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
		StepID timebox.ID `json:"step_id"`
		Token  Token      `json:"token"`
		Error  string     `json:"error"`
	}

	RetryScheduledEvent struct {
		FlowID      timebox.ID `json:"flow_id"`
		StepID      timebox.ID `json:"step_id"`
		Token       Token      `json:"token"`
		RetryCount  int        `json:"retry_count"`
		NextRetryAt time.Time  `json:"next_retry_at"`
		Error       string     `json:"error"`
	}

	WorkflowActivatedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
	}

	WorkflowDeactivatedEvent struct {
		FlowID timebox.ID `json:"flow_id"`
	}
)

const (
	EventTypeStepRegistered      timebox.EventType = "step_registered"
	EventTypeStepUnregistered    timebox.EventType = "step_unregistered"
	EventTypeStepHealthChanged   timebox.EventType = "step_health_changed"
	EventTypeWorkflowActivated   timebox.EventType = "workflow_activated"
	EventTypeWorkflowDeactivated timebox.EventType = "workflow_deactivated"
	EventTypeWorkflowStarted     timebox.EventType = "workflow_started"
	EventTypeWorkflowCompleted   timebox.EventType = "workflow_completed"
	EventTypeWorkflowFailed      timebox.EventType = "workflow_failed"
	EventTypeAttributeSet        timebox.EventType = "attribute_set"
	EventTypeStepStarted         timebox.EventType = "step_started"
	EventTypeStepCompleted       timebox.EventType = "step_completed"
	EventTypeStepFailed          timebox.EventType = "step_failed"
	EventTypeStepSkipped         timebox.EventType = "step_skipped"
	EventTypeWorkStarted         timebox.EventType = "work_started"
	EventTypeWorkCompleted       timebox.EventType = "work_completed"
	EventTypeWorkFailed          timebox.EventType = "work_failed"
	EventTypeRetryScheduled      timebox.EventType = "retry_scheduled"
)
