package engine

import (
	"log/slog"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
)

// ScheduleTimeout schedules a timeout for a step
func (e *Engine) ScheduleTimeout(flowID api.FlowID, stepID api.StepID,
	firesAt time.Time, attrs []api.Name,
	upstreamStepIDs []api.StepID) error {
	if err := e.flowTx(flowID, func(tx *flowTx) error {
		return events.Raise(tx.FlowAggregator,
			api.EventTypeTimeoutScheduled,
			api.TimeoutScheduledEvent{
				FlowID:          flowID,
				StepID:          stepID,
				FiresAt:         firesAt,
				Attributes:      attrs,
				UpstreamStepIDs: upstreamStepIDs,
			},
		)
	}); err != nil {
		return err
	}

	if err := e.raisePartitionEvent(api.EventTypeTimeoutScheduled,
		api.TimeoutScheduledEvent{
			FlowID:          flowID,
			StepID:          stepID,
			FiresAt:         firesAt,
			Attributes:      attrs,
			UpstreamStepIDs: upstreamStepIDs,
		},
	); err != nil {
		return err
	}

	e.RegisterTask(e.timeoutTask, firesAt)
	return nil
}

// CancelTimeout removes a timeout for a step
func (e *Engine) CancelTimeout(flowID api.FlowID, stepID api.StepID) error {
	if err := e.flowTx(flowID, func(tx *flowTx) error {
		return events.Raise(tx.FlowAggregator,
			api.EventTypeTimeoutCanceled,
			api.TimeoutCanceledEvent{
				FlowID: flowID,
				StepID: stepID,
			},
		)
	}); err != nil {
		return err
	}

	return e.raisePartitionEvent(api.EventTypeTimeoutCanceled,
		api.TimeoutCanceledEvent{
			FlowID: flowID,
			StepID: stepID,
		},
	)
}

// fireExpiredTimeouts checks for expired timeouts and raises events
func (e *Engine) fireExpiredTimeouts() error {
	var expired []*api.TimeoutEntry

	cmd := func(_ *api.PartitionState,
		ag *PartitionAggregator) error {
		part := ag.Value()
		now := time.Now()

		for len(part.Timeouts) > 0 {
			entry := part.Timeouts[0]
			if entry.FiresAt.After(now) {
				break
			}

			expired = append(expired, entry)

			if err := events.Raise(ag, api.EventTypeTimeoutFired,
				api.TimeoutFiredEvent{
					FlowID:     entry.FlowID,
					StepID:     entry.StepID,
					FiresAt:    entry.FiresAt,
					Attributes: []api.Name{},
				},
			); err != nil {
				slog.Error("Failed to raise timeout fired",
					log.FlowID(entry.FlowID),
					log.StepID(entry.StepID),
					log.Error(err))
				return err
			}

			part = ag.Value()
		}
		return nil
	}

	if _, err := e.execPartition(cmd); err != nil {
		return err
	}

	for _, entry := range expired {
		if err := e.onTimeoutFired(entry.FlowID, entry.StepID); err != nil {
			slog.Error("Failed to handle timeout",
				log.FlowID(entry.FlowID),
				log.StepID(entry.StepID),
				log.Error(err))
		}
	}
	return nil
}
