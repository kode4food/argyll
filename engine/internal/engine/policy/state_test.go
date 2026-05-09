package policy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestStepStatusPolicy(t *testing.T) {
	assert.True(t, policy.StepPending(api.StepPending))
	assert.True(t, policy.StepActive(api.StepActive))
	assert.True(t, policy.StepFailed(api.StepFailed))

	assert.True(t, policy.StepComplete(api.StepCompleted))
	assert.True(t, policy.StepComplete(api.StepSkipped))
	assert.False(t, policy.StepComplete(api.StepFailed))

	assert.True(t, policy.StepSucceeded(api.StepCompleted))
	assert.False(t, policy.StepSucceeded(api.StepSkipped))
	assert.True(t, policy.StepTerminal(api.StepCompleted))
	assert.True(t, policy.StepTerminal(api.StepFailed))
}

func TestWorkStatusPolicy(t *testing.T) {
	assert.True(t, policy.WorkActive(api.WorkActive))
	assert.False(t, policy.WorkActive(api.WorkPending))

	assert.True(t, policy.WorkBlocksFlowDeactivation(api.WorkPending))
	assert.True(t, policy.WorkBlocksFlowDeactivation(api.WorkActive))
	assert.False(t, policy.WorkBlocksFlowDeactivation(api.WorkSucceeded))

	assert.True(t, policy.WorkTerminal(api.WorkSucceeded))
	assert.True(t, policy.WorkTerminal(api.WorkFailed))
	assert.False(t, policy.WorkTerminal(api.WorkNotCompleted))

	assert.True(t, policy.WorkClaimableForRetry(api.WorkPending))
	assert.True(t, policy.WorkClaimableForRetry(api.WorkFailed))
	assert.False(t, policy.WorkClaimableForRetry(api.WorkActive))
}

func TestTransitionPolicy(t *testing.T) {
	assert.True(t, policy.FlowTerminal(api.FlowCompleted))
	assert.False(t, policy.FlowTerminal(api.FlowActive))

	assert.True(t, policy.WorkCanTransition(api.WorkPending, api.WorkActive))
	assert.False(t, policy.WorkCanTransition(api.WorkSucceeded, api.WorkActive))
}
