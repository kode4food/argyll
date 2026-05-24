package api

import "time"

type (
	// StepRegisteredEvent is emitted when a step is registered with the engine
	StepRegisteredEvent struct {
		Step *Step `json:"step"`
	}

	// StepUnregisteredEvent is emitted when a step is removed from the engine
	StepUnregisteredEvent struct {
		StepID StepID `json:"step_id"`
	}

	// StepUpdatedEvent is emitted when a step definition is modified
	StepUpdatedEvent struct {
		Step *Step `json:"step"`
	}

	// StepHealthChangedEvent is emitted when a step's health status changes
	StepHealthChangedEvent struct {
		NodeID NodeID       `json:"node_id"`
		StepID StepID       `json:"step_id"`
		Status HealthStatus `json:"status"`
		Error  string       `json:"error,omitempty"`
	}

	// FlowStartedEvent is emitted when a flow execution begins
	FlowStartedEvent struct {
		Plan     *ExecutionPlan `json:"plan"`
		Init     InitArgs       `json:"init"`
		Metadata Metadata       `json:"metadata,omitempty"`
		Labels   Labels         `json:"labels,omitempty"`
		FlowID   FlowID         `json:"flow_id"`
	}

	// FlowCompletedEvent is emitted when a flow completes successfully
	FlowCompletedEvent struct {
		Result Args   `json:"result"`
		FlowID FlowID `json:"flow_id"`
	}

	// FlowFailedEvent is emitted when a flow fails
	FlowFailedEvent struct {
		FlowID FlowID `json:"flow_id"`
		Error  string `json:"error"`
	}

	// StepStartedEvent is emitted when a step begins execution
	StepStartedEvent struct {
		Inputs    Args           `json:"inputs"`
		WorkItems map[Token]Args `json:"work_items"`
		FlowID    FlowID         `json:"flow_id"`
		StepID    StepID         `json:"step_id"`
	}

	// StepCompletedEvent is emitted when a step completes successfully
	StepCompletedEvent struct {
		Outputs  Args   `json:"outputs"`
		FlowID   FlowID `json:"flow_id"`
		StepID   StepID `json:"step_id"`
		Duration int64  `json:"duration"`
	}

	// StepFailedEvent is emitted when a step fails
	StepFailedEvent struct {
		FlowID      FlowID `json:"flow_id"`
		StepID      StepID `json:"step_id"`
		Error       string `json:"error"`
		Inputs      Args   `json:"inputs,omitempty"`
		Unsatisfied []Name `json:"unsatisfied,omitempty"`
	}

	// StepSkippedEvent is emitted when a step is skipped due to predicate
	StepSkippedEvent struct {
		FlowID      FlowID `json:"flow_id"`
		StepID      StepID `json:"step_id"`
		Reason      string `json:"reason"`
		Inputs      Args   `json:"inputs,omitempty"`
		Unsatisfied []Name `json:"unsatisfied,omitempty"`
	}

	// AttributeSetEvent is emitted when a flow attribute value is set
	AttributeSetEvent struct {
		Value  any    `json:"value"`
		FlowID FlowID `json:"flow_id"`
		StepID StepID `json:"step_id"`
		Key    Name   `json:"key"`
	}

	// WorkStartedEvent is emitted when a work item begins execution
	WorkStartedEvent struct {
		Inputs Args   `json:"inputs"`
		FlowID FlowID `json:"flow_id"`
		StepID StepID `json:"step_id"`
		Token  Token  `json:"token"`
	}

	// WorkSucceededEvent is emitted when a work item succeeds
	WorkSucceededEvent struct {
		Outputs Args   `json:"outputs"`
		FlowID  FlowID `json:"flow_id"`
		StepID  StepID `json:"step_id"`
		Token   Token  `json:"token"`
	}

	// WorkFailedEvent is emitted when a work item fails permanently
	WorkFailedEvent struct {
		FlowID FlowID `json:"flow_id"`
		StepID StepID `json:"step_id"`
		Token  Token  `json:"token"`
		Error  string `json:"error"`
	}

	// WorkNotCompletedEvent is emitted when a work item fails transiently
	WorkNotCompletedEvent struct {
		FlowID FlowID `json:"flow_id"`
		StepID StepID `json:"step_id"`
		Token  Token  `json:"token"`
		Error  string `json:"error"`
	}

	// WorkRetryScheduledEvent is emitted when a failed work item is to be
	// retried
	WorkRetryScheduledEvent struct {
		NextRetryAt time.Time `json:"next_retry_at"`
		FlowID      FlowID    `json:"flow_id"`
		StepID      StepID    `json:"step_id"`
		Token       Token     `json:"token"`
		Error       string    `json:"error"`
		RetryCount  int       `json:"retry_count"`
	}

	// DispatchDeferredEvent is emitted when runnable work cannot be started
	// immediately and must be picked up later
	DispatchDeferredEvent struct {
		FlowID FlowID `json:"flow_id"`
		StepID StepID `json:"step_id"`
	}

	// FlowDeactivatedEvent is emitted when a flow becomes inactive
	FlowDeactivatedEvent struct {
		FlowID FlowID     `json:"flow_id"`
		Status FlowStatus `json:"status"`
	}

	// CompStartedEvent is emitted when compensation begins for a work item
	CompStartedEvent struct {
		FlowID FlowID `json:"flow_id"`
		StepID StepID `json:"step_id"`
		Token  Token  `json:"token"`
	}

	// CompRetryScheduledEvent is emitted when a compensation is to be retried
	CompRetryScheduledEvent struct {
		NextRetryAt time.Time `json:"next_retry_at"`
		FlowID      FlowID    `json:"flow_id"`
		StepID      StepID    `json:"step_id"`
		Token       Token     `json:"token"`
		Error       string    `json:"error"`
		RetryCount  int       `json:"retry_count"`
	}

	// CompSucceededEvent is emitted when compensation succeeds
	CompSucceededEvent struct {
		FlowID FlowID `json:"flow_id"`
		StepID StepID `json:"step_id"`
		Token  Token  `json:"token"`
	}

	// CompFailedEvent is emitted when compensation fails permanently
	CompFailedEvent struct {
		FlowID FlowID `json:"flow_id"`
		StepID StepID `json:"step_id"`
		Token  Token  `json:"token"`
		Error  string `json:"error"`
	}

	EventType string
)

const (
	EventTypeStepRegistered     EventType = "step_registered"
	EventTypeStepUnregistered   EventType = "step_unregistered"
	EventTypeStepUpdated        EventType = "step_updated"
	EventTypeStepHealthChanged  EventType = "step_health_changed"
	EventTypeFlowDeactivated    EventType = "flow_deactivated"
	EventTypeFlowStarted        EventType = "flow_started"
	EventTypeFlowCompleted      EventType = "flow_completed"
	EventTypeFlowFailed         EventType = "flow_failed"
	EventTypeAttributeSet       EventType = "attribute_set"
	EventTypeStepStarted        EventType = "step_started"
	EventTypeStepCompleted      EventType = "step_completed"
	EventTypeStepFailed         EventType = "step_failed"
	EventTypeStepSkipped        EventType = "step_skipped"
	EventTypeWorkStarted        EventType = "work_started"
	EventTypeWorkSucceeded      EventType = "work_succeeded"
	EventTypeWorkFailed         EventType = "work_failed"
	EventTypeWorkNotCompleted   EventType = "work_not_completed"
	EventTypeWorkRetryScheduled EventType = "work_retry_scheduled"
	EventTypeDispatchDeferred   EventType = "dispatch_deferred"
	EventTypeCompStarted        EventType = "comp_started"
	EventTypeCompRetryScheduled EventType = "comp_retry_scheduled"
	EventTypeCompSucceeded      EventType = "comp_succeeded"
	EventTypeCompFailed         EventType = "comp_failed"
)
