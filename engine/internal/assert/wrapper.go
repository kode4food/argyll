package assert

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	Getter interface {
		GetAttribute(
			ctx context.Context, flowID api.FlowID, attr api.Name,
		) (any, bool, error)
	}

	// Wrapper wraps testify assertions with Argyll-specific helpers
	Wrapper struct {
		*testing.T
		*assert.Assertions
		Require *assert.Assertions
	}
)

// DefaultRetryInterval is the default polling interval for Eventually checks
const DefaultRetryInterval = 100 * time.Millisecond

// New creates a new test assertion wrapper with both assert and require from
// testify plus Argyll-specific helpers
func New(t *testing.T) *Wrapper {
	return &Wrapper{
		T:          t,
		Assertions: assert.New(t),
		Require:    assert.New(t),
	}
}

// StepValid asserts that a step is valid
func (w *Wrapper) StepValid(t *api.Step) {
	w.Helper()
	w.NoError(t.Validate())
	w.NotEmpty(t.ID)
	w.NotEmpty(t.Name)

	switch t.Type {
	case api.StepTypeSync, api.StepTypeAsync:
		w.NotNil(t.HTTP, "HTTP steps should have HTTPConfig")
		if t.HTTP != nil {
			w.NotEmpty(t.HTTP.Endpoint)
		}
	case api.StepTypeScript:
		w.NotNil(t.Script, "Script steps should have ScriptConfig")
		if t.Script != nil {
			w.NotEmpty(t.Script.Language)
			w.NotEmpty(t.Script.Script)
		}
	}
}

// StepInvalid asserts that a step is invalid and returns the validation error
func (w *Wrapper) StepInvalid(
	t *api.Step, expectedErrorContains string,
) error {
	w.Helper()
	err := t.Validate()
	w.Error(err)
	if err != nil && expectedErrorContains != "" {
		w.Contains(err.Error(), expectedErrorContains)
	}
	return err
}

// FlowStatus asserts the status of a flow
func (w *Wrapper) FlowStatus(flow *api.FlowState, expected api.FlowStatus) {
	w.Helper()
	w.Equal(expected, flow.Status)
}

// FlowHasState asserts that a flow has specific state keys
func (w *Wrapper) FlowHasState(
	ctx context.Context, get Getter, flowID api.FlowID, keys ...api.Name,
) {
	w.Helper()
	for _, key := range keys {
		_, ok, err := get.GetAttribute(ctx, flowID, key)
		w.NoError(err, "failed to check state key: %s", key)
		w.True(ok, "flow should have state key: %s", key)
	}
}

// FlowStateEquals asserts that a state key has the expected value
func (w *Wrapper) FlowStateEquals(
	ctx context.Context, get Getter, flowID api.FlowID, key api.Name,
	expected any,
) {
	w.Helper()
	val, ok, err := get.GetAttribute(ctx, flowID, key)
	w.NoError(err, "failed to get state key: %s", key)
	w.True(ok, "flow should have state key: %s", key)
	w.Equal(expected, val)
}

// ConfigValid asserts that a configuration is valid
func (w *Wrapper) ConfigValid(cfg *config.Config) {
	w.Helper()
	w.NoError(cfg.Validate())
	w.True(cfg.APIPort > 0 && cfg.APIPort <= 65535)
	w.True(cfg.StepTimeout > 0)
}

// ConfigInvalid asserts that a configuration is invalid
func (w *Wrapper) ConfigInvalid(cfg *config.Config, contains string) {
	w.Helper()
	err := cfg.Validate()
	w.Error(err)
	if contains != "" {
		w.Contains(err.Error(), contains)
	}
}

// Eventually runs a condition repeatedly until it passes or times out
func (w *Wrapper) Eventually(
	condition func() bool, timeout time.Duration, msg string, args ...any,
) {
	w.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(DefaultRetryInterval)
	}
	w.Fail(msg, args...)
}

// EventuallyWithError runs a condition that returns an error until it succeeds
// or times out
func (w *Wrapper) EventuallyWithError(
	condition func() error, timeout time.Duration, msg string, args ...any,
) {
	w.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		err := condition()
		if err == nil {
			return
		}
		lastErr = err
		time.Sleep(DefaultRetryInterval)
	}
	if lastErr != nil {
		w.Fail(msg+": last error: "+lastErr.Error(), args...)
		return
	}
	w.Fail(msg, args...)
}
