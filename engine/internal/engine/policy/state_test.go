package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestStepStatusPolicy(t *testing.T) {
	assert.True(t, StepPending(api.StepPending))
	assert.True(t, StepActive(api.StepActive))
	assert.True(t, StepFailed(api.StepFailed))

	assert.True(t, StepComplete(api.StepCompleted))
	assert.True(t, StepComplete(api.StepSkipped))
	assert.False(t, StepComplete(api.StepFailed))

	assert.True(t, StepSucceeded(api.StepCompleted))
	assert.False(t, StepSucceeded(api.StepSkipped))
	assert.True(t, StepTerminal(api.StepCompleted))
	assert.True(t, StepTerminal(api.StepFailed))
}

func TestWorkStatusPolicy(t *testing.T) {
	assert.True(t, WorkActive(api.WorkActive))
	assert.False(t, WorkActive(api.WorkPending))

	assert.True(t, WorkBlocksFlowDeactivation(api.WorkPending))
	assert.True(t, WorkBlocksFlowDeactivation(api.WorkActive))
	assert.False(t, WorkBlocksFlowDeactivation(api.WorkSucceeded))

	assert.True(t, WorkTerminal(api.WorkSucceeded))
	assert.True(t, WorkTerminal(api.WorkFailed))
	assert.False(t, WorkTerminal(api.WorkNotCompleted))

	assert.True(t, WorkClaimableForRetry(api.WorkPending))
	assert.True(t, WorkClaimableForRetry(api.WorkFailed))
	assert.False(t, WorkClaimableForRetry(api.WorkActive))
}

func TestTransitionPolicy(t *testing.T) {
	assert.True(t, FlowTerminal(api.FlowCompleted))
	assert.False(t, FlowTerminal(api.FlowActive))

	assert.True(t, WorkCanTransition(api.WorkPending, api.WorkActive))
	assert.False(t, WorkCanTransition(api.WorkSucceeded, api.WorkActive))
}
