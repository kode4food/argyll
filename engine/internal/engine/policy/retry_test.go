package policy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRecoverableDeadline(t *testing.T) {
	now := time.Unix(10, 0)
	later := time.Unix(20, 0)

	at, ok := RecoverableDeadline(
		api.ExecutionState{Status: api.StepActive},
		api.WorkState{Status: api.WorkActive},
		now,
	)
	assert.True(t, ok)
	assert.Equal(t, now, at)

	at, ok = RecoverableDeadline(
		api.ExecutionState{Status: api.StepPending},
		api.WorkState{Status: api.WorkPending, NextRetryAt: later},
		now,
	)
	assert.True(t, ok)
	assert.Equal(t, later, at)

	at, ok = RecoverableDeadline(
		api.ExecutionState{Status: api.StepActive},
		api.WorkState{Status: api.WorkPending},
		now,
	)
	assert.True(t, ok)
	assert.Equal(t, now, at)

	_, ok = RecoverableDeadline(
		api.ExecutionState{Status: api.StepPending},
		api.WorkState{Status: api.WorkFailed},
		now,
	)
	assert.False(t, ok)
}

func TestRetryStartDecision(t *testing.T) {
	now := time.Unix(10, 0)
	later := time.Unix(20, 0)

	action, at := RetryStartDecision(
		api.WorkState{Status: api.WorkPending, NextRetryAt: later},
		now,
	)
	assert.Equal(t, RetryStartWait, action)
	assert.Equal(t, later, at)

	action, _ = RetryStartDecision(api.WorkState{Status: api.WorkPending}, now)
	assert.Equal(t, RetryStartCheckPending, action)

	action, _ = RetryStartDecision(
		api.WorkState{Status: api.WorkFailed, NextRetryAt: now.Add(-time.Second)},
		now,
	)
	assert.Equal(t, RetryStartNow, action)

	action, at = RetryStartDecision(api.WorkState{Status: api.WorkFailed}, now)
	assert.Equal(t, RetryStartWait, action)
	assert.True(t, at.IsZero())

	action, _ = RetryStartDecision(api.WorkState{Status: api.WorkSucceeded}, now)
	assert.Equal(t, RetryStartIgnore, action)
}

func TestRecoverable(t *testing.T) {
	assert.True(t, Recoverable(
		api.ExecutionState{Status: api.StepActive},
		api.WorkState{Status: api.WorkPending},
	))
	assert.False(t, Recoverable(
		api.ExecutionState{Status: api.StepPending},
		api.WorkState{Status: api.WorkSucceeded},
	))
}
