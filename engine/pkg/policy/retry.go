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
	// the returned time, which is the work item's pending retry timestamp
	RetryStartWait

	// RetryStartCheckPending means pending work may start only after normal
	// predicate and parallelism checks are applied by the executor
	RetryStartCheckPending
)

// RetryStartDecision classifies retry task handling for a work item at the
// supplied time. It keeps the state/timestamp policy separate from executor
// concerns such as predicates, dispatch locality, and event raising
func RetryStartDecision(
	work api.WorkState, when time.Time,
) (RetryStartAction, time.Time) {
	if !WorkPending(work.Status) {
		return RetryStartIgnore, time.Time{}
	}
	if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(when) {
		return RetryStartWait, work.NextRetryAt
	}
	return RetryStartCheckPending, time.Time{}
}

// RecoverableDeadline returns when recovery should schedule a work item for
// retry handling. Only pending work qualifies, at NextRetryAt when one exists,
// with active-step pending work also recoverable immediately
func RecoverableDeadline(
	ex api.ExecutionState, work api.WorkState, when time.Time,
) (time.Time, bool) {
	if !WorkPending(work.Status) {
		return time.Time{}, false
	}
	if !work.NextRetryAt.IsZero() {
		return work.NextRetryAt, true
	}
	if ex.Status == api.StepActive {
		return when, true
	}
	return time.Time{}, false
}

// CompRetryAt returns when a pending compensation should next be attempted, and
// whether it is waiting for one. A lapsed time means attempt it now
func CompRetryAt(work api.WorkState, when time.Time) (time.Time, bool) {
	if !WorkCompPending(work.Status) {
		return time.Time{}, false
	}
	if work.NextRetryAt.After(when) {
		return work.NextRetryAt, true
	}
	return when, true
}
