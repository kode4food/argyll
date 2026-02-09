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
		StepID StepID       `json:"step_id"`
		Status HealthStatus `json:"status"`
		Error  string       `json:"error,omitempty"`
	}

	// FlowStartedEvent is emitted when a flow execution begins
	FlowStartedEvent struct {
		Plan     *ExecutionPlan `json:"plan"`
		Init     Args           `json:"init"`
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
		FlowID    FlowID         `json:"flow_id"`
		StepID    StepID         `json:"step_id"`
		WorkItems map[Token]Args `json:"work_items"`
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
		FlowID FlowID `json:"flow_id"`
		StepID StepID `json:"step_id"`
		Error  string `json:"error"`
	}

	// StepSkippedEvent is emitted when a step is skipped due to predicate
	StepSkippedEvent struct {
		FlowID FlowID `json:"flow_id"`
		StepID StepID `json:"step_id"`
		Reason string `json:"reason"`
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
		FlowID FlowID `json:"flow_id"`
		StepID StepID `json:"step_id"`
		Token  Token  `json:"token"`
		Inputs Args   `json:"inputs"`
	}

	// WorkSucceededEvent is emitted when a work item succeeds
	WorkSucceededEvent struct {
		FlowID  FlowID `json:"flow_id"`
		StepID  StepID `json:"step_id"`
		Token   Token  `json:"token"`
		Outputs Args   `json:"outputs"`
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
		FlowID     FlowID `json:"flow_id"`
		StepID     StepID `json:"step_id"`
		Token      Token  `json:"token"`
		RetryToken Token  `json:"retry_token,omitempty"`
		Error      string `json:"error"`
	}

	// RetryScheduledEvent is emitted when a failed work item is to be retried
	RetryScheduledEvent struct {
		FlowID      FlowID    `json:"flow_id"`
		StepID      StepID    `json:"step_id"`
		Token       Token     `json:"token"`
		RetryCount  int       `json:"retry_count"`
		NextRetryAt time.Time `json:"next_retry_at"`
		Error       string    `json:"error"`
	}

	// FlowActivatedEvent is emitted when a flow becomes active
	FlowActivatedEvent struct {
		FlowID       FlowID `json:"flow_id"`
		ParentFlowID FlowID `json:"parent_flow_id,omitempty"`
		Labels       Labels `json:"labels,omitempty"`
	}

	// FlowDigestUpdatedEvent is emitted when a flow summary changes
	FlowDigestUpdatedEvent struct {
		FlowID      FlowID     `json:"flow_id"`
		Status      FlowStatus `json:"status"`
		CompletedAt time.Time  `json:"completed_at"`
		Error       string     `json:"error,omitempty"`
	}

	// FlowDeactivatedEvent is emitted when a flow becomes inactive
	FlowDeactivatedEvent struct {
		FlowID FlowID `json:"flow_id"`
	}

	// FlowArchivingEvent is emitted when a flow is scheduled for archiving
	FlowArchivingEvent struct {
		FlowID FlowID `json:"flow_id"`
	}

	// FlowArchivedEvent is emitted when a flow is archived
	FlowArchivedEvent struct {
		FlowID FlowID `json:"flow_id"`
	}

	EventType string
)

const (
	EventTypeStepRegistered    EventType = "step_registered"
	EventTypeStepUnregistered  EventType = "step_unregistered"
	EventTypeStepUpdated       EventType = "step_updated"
	EventTypeStepHealthChanged EventType = "step_health_changed"
	EventTypeFlowActivated     EventType = "flow_activated"
	EventTypeFlowDeactivated   EventType = "flow_deactivated"
	EventTypeFlowArchiving     EventType = "flow_archiving"
	EventTypeFlowArchived      EventType = "flow_archived"
	EventTypeFlowDigestUpdated EventType = "flow_digest_updated"
	EventTypeFlowStarted       EventType = "flow_started"
	EventTypeFlowCompleted     EventType = "flow_completed"
	EventTypeFlowFailed        EventType = "flow_failed"
	EventTypeAttributeSet      EventType = "attribute_set"
	EventTypeStepStarted       EventType = "step_started"
	EventTypeStepCompleted     EventType = "step_completed"
	EventTypeStepFailed        EventType = "step_failed"
	EventTypeStepSkipped       EventType = "step_skipped"
	EventTypeWorkStarted       EventType = "work_started"
	EventTypeWorkSucceeded     EventType = "work_succeeded"
	EventTypeWorkFailed        EventType = "work_failed"
	EventTypeWorkNotCompleted  EventType = "work_not_completed"
	EventTypeRetryScheduled    EventType = "retry_scheduled"
)
