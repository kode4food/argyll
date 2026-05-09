package policy_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRecoverableDeadline(t *testing.T) {
	now := time.Unix(10, 0)
	later := time.Unix(20, 0)

	at, ok := policy.RecoverableDeadline(
		api.ExecutionState{Status: api.StepActive},
		api.WorkState{Status: api.WorkActive},
		now,
	)
	assert.True(t, ok)
	assert.Equal(t, now, at)

	at, ok = policy.RecoverableDeadline(
		api.ExecutionState{Status: api.StepPending},
		api.WorkState{Status: api.WorkPending, NextRetryAt: later},
		now,
	)
	assert.True(t, ok)
	assert.Equal(t, later, at)

	at, ok = policy.RecoverableDeadline(
		api.ExecutionState{Status: api.StepActive},
		api.WorkState{Status: api.WorkPending},
		now,
	)
	assert.True(t, ok)
	assert.Equal(t, now, at)

	_, ok = policy.RecoverableDeadline(
		api.ExecutionState{Status: api.StepPending},
		api.WorkState{Status: api.WorkFailed},
		now,
	)
	assert.False(t, ok)
}

func TestRetryStartDecision(t *testing.T) {
	now := time.Unix(10, 0)
	later := time.Unix(20, 0)

	action, at := policy.RetryStartDecision(
		api.WorkState{Status: api.WorkPending, NextRetryAt: later},
		now,
	)
	assert.Equal(t, policy.RetryStartWait, action)
	assert.Equal(t, later, at)

	action, _ = policy.RetryStartDecision(
		api.WorkState{Status: api.WorkPending}, now,
	)
	assert.Equal(t, policy.RetryStartCheckPending, action)

	action, _ = policy.RetryStartDecision(
		api.WorkState{
			Status:      api.WorkFailed,
			NextRetryAt: now.Add(-time.Second),
		},
		now,
	)
	assert.Equal(t, policy.RetryStartNow, action)

	action, at = policy.RetryStartDecision(
		api.WorkState{Status: api.WorkFailed}, now,
	)
	assert.Equal(t, policy.RetryStartWait, action)
	assert.True(t, at.IsZero())

	action, _ = policy.RetryStartDecision(
		api.WorkState{Status: api.WorkSucceeded}, now,
	)
	assert.Equal(t, policy.RetryStartIgnore, action)
}

func TestRecoverable(t *testing.T) {
	assert.True(t, policy.Recoverable(
		api.ExecutionState{Status: api.StepActive},
		api.WorkState{Status: api.WorkPending},
	))
	assert.False(t, policy.Recoverable(
		api.ExecutionState{Status: api.StepPending},
		api.WorkState{Status: api.WorkSucceeded},
	))
}
