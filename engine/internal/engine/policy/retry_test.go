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

	_, ok := policy.RecoverableDeadline(
		api.ExecutionState{Status: api.StepActive},
		api.WorkState{Status: api.WorkActive},
		now,
	)
	assert.False(t, ok)

	at, ok := policy.RecoverableDeadline(
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
		api.WorkState{Status: api.WorkFailed, NextRetryAt: later},
		now,
	)
	assert.False(t, ok)

	_, ok = policy.RecoverableDeadline(
		api.ExecutionState{Status: api.StepActive},
		api.WorkState{Status: api.WorkNotCompleted},
		now,
	)
	assert.False(t, ok)

	_, ok = policy.RecoverableDeadline(
		api.ExecutionState{Status: api.StepPending},
		api.WorkState{Status: api.WorkPending},
		now,
	)
	assert.False(t, ok)
}

func TestRetryStartDecision(t *testing.T) {
	now := time.Unix(10, 0)
	later := time.Unix(20, 0)

	action, at := policy.RetryStartDecision(
		api.WorkState{Status: api.WorkPending, NextRetryAt: later}, now,
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
	assert.Equal(t, policy.RetryStartIgnore, action)

	action, _ = policy.RetryStartDecision(
		api.WorkState{Status: api.WorkSucceeded}, now,
	)
	assert.Equal(t, policy.RetryStartIgnore, action)

	action, _ = policy.RetryStartDecision(
		api.WorkState{Status: api.WorkActive}, now,
	)
	assert.Equal(t, policy.RetryStartIgnore, action)

	// Not-completed work is settled by the same command that reports it, so
	// retry handling never finds one to restart
	action, _ = policy.RetryStartDecision(
		api.WorkState{Status: api.WorkNotCompleted}, now,
	)
	assert.Equal(t, policy.RetryStartIgnore, action)
}

func TestCompRetryAt(t *testing.T) {
	now := time.Unix(10, 0)
	later := time.Unix(20, 0)

	at, ok := policy.CompRetryAt(
		api.WorkState{Status: api.WorkCompPending, NextRetryAt: later}, now,
	)
	assert.True(t, ok)
	assert.Equal(t, later, at)

	// A lapsed or unset time means attempt it now
	at, ok = policy.CompRetryAt(
		api.WorkState{Status: api.WorkCompPending}, now,
	)
	assert.True(t, ok)
	assert.Equal(t, now, at)

	_, ok = policy.CompRetryAt(
		api.WorkState{Status: api.WorkCompensating}, now,
	)
	assert.False(t, ok)

	_, ok = policy.CompRetryAt(
		api.WorkState{Status: api.WorkCompensated}, now,
	)
	assert.False(t, ok)
}
