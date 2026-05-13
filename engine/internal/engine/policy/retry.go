package policy

import (
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// RetryStartAction describes what retry task handling should do with a work
// item when its scheduled retry callback fires
type RetryStartAction int

const (
	// RetryStartIgnore means the work item is already in a state that retry
	// handling should not touch
	RetryStartIgnore RetryStartAction = iota

	// RetryStartWait means retry handling should schedule another callback at
	// the returned time. A zero time preserves existing behavior for failed
	// work without a retry timestamp: do not start and do not reschedule
	RetryStartWait

	// RetryStartCheckPending means pending work may start only after normal
	// predicate and parallelism checks are applied by the executor
	RetryStartCheckPending

	// RetryStartNow means the work item should be restarted immediately,
	// subject only to the executor successfully raising WorkStarted
	RetryStartNow
)

// RetryStartDecision classifies retry task handling for a work item at the
// supplied time. It keeps the state/timestamp policy separate from executor
// concerns such as predicates, dispatch locality, and event raising
func RetryStartDecision(
	work api.WorkState, when time.Time,
) (RetryStartAction, time.Time) {
	switch work.Status {
	case api.WorkPending:
		if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(when) {
			return RetryStartWait, work.NextRetryAt
		}
		return RetryStartCheckPending, time.Time{}
	case api.WorkFailed:
		if work.NextRetryAt.IsZero() || work.NextRetryAt.After(when) {
			return RetryStartWait, work.NextRetryAt
		}
		return RetryStartNow, time.Time{}
	case api.WorkActive, api.WorkNotCompleted:
		return RetryStartNow, time.Time{}
	default:
		return RetryStartIgnore, time.Time{}
	}
}

// RecoverableDeadline returns when startup recovery should schedule a work
// item for retry handling. Active and not-completed work are recovered now;
// pending or failed work are recovered at NextRetryAt when one exists, with
// active-step pending work also recoverable immediately
func RecoverableDeadline(
	exec api.ExecutionState, work api.WorkState, when time.Time,
) (time.Time, bool) {
	switch work.Status {
	case api.WorkActive, api.WorkNotCompleted:
		return when, true
	case api.WorkPending:
		if !work.NextRetryAt.IsZero() {
			return work.NextRetryAt, true
		}
		if exec.Status == api.StepActive {
			return when, true
		}
		return time.Time{}, false
	case api.WorkFailed:
		if !work.NextRetryAt.IsZero() {
			return work.NextRetryAt, true
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}

// Recoverable reports whether startup recovery should consider this work item
func Recoverable(exec api.ExecutionState, work api.WorkState) bool {
	_, ok := RecoverableDeadline(exec, work, time.Time{})
	return ok
}
