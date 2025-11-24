package assert

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kode4food/spuds/engine/internal/config"
	"github.com/kode4food/spuds/engine/pkg/api"
)

type mockGetter struct {
	attrs map[api.FlowID]map[api.Name]any
	err   error
}

func (g *mockGetter) GetAttribute(
	_ context.Context, flowID api.FlowID, attr api.Name,
) (any, bool, error) {
	if g.err != nil {
		return nil, false, g.err
	}
	if flowAttrs, ok := g.attrs[flowID]; ok {
		if val, ok := flowAttrs[attr]; ok {
			return val, true, nil
		}
	}
	return nil, false, nil
}

func TestNew(t *testing.T) {
	wrapper := New(t)

	if wrapper.T != t {
		t.Error("Wrapper.T should be set to the testing.T instance")
	}
	if wrapper.Assertions == nil {
		t.Error("Wrapper.Assertions should be initialized")
	}
	if wrapper.Require == nil {
		t.Error("Wrapper.Require should be initialized")
	}
}

func TestStepValid(t *testing.T) {
	tests := []struct {
		name string
		step *api.Step
	}{
		{
			name: "valid sync step",
			step: &api.Step{
				ID:      "test-sync",
				Name:    "Test Sync",
				Version: "1.0.0",
				Type:    api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost/test",
					Timeout:  1000,
				},
			},
		},
		{
			name: "valid async step",
			step: &api.Step{
				ID:      "test-async",
				Name:    "Test Async",
				Version: "1.0.0",
				Type:    api.StepTypeAsync,
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost/test",
					Timeout:  1000,
				},
			},
		},
		{
			name: "valid script step with Ale",
			step: &api.Step{
				ID:      "test-script",
				Name:    "Test Script",
				Version: "1.0.0",
				Type:    api.StepTypeScript,
				Script: &api.ScriptConfig{
					Language: api.ScriptLangAle,
					Script:   "(+ 1 2)",
				},
			},
		},
		{
			name: "valid script step with Lua",
			step: &api.Step{
				ID:      "test-lua",
				Name:    "Test Lua",
				Version: "1.0.0",
				Type:    api.StepTypeScript,
				Script: &api.ScriptConfig{
					Language: api.ScriptLangLua,
					Script:   "return {result = 42}",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := New(t)
			w.StepValid(tt.step)
		})
	}
}

func TestStepInvalid(t *testing.T) {
	tests := []struct {
		name                 string
		step                 *api.Step
		expectedErrorContain string
	}{
		{
			name: "missing ID",
			step: &api.Step{
				Name:    "Test",
				Version: "1.0.0",
				Type:    api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost/test",
				},
			},
			expectedErrorContain: "ID",
		},
		{
			name: "missing Name",
			step: &api.Step{
				ID:      "test-id",
				Version: "1.0.0",
				Type:    api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost/test",
				},
			},
			expectedErrorContain: "name",
		},
		{
			name: "missing Version",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test",
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost/test",
				},
			},
			expectedErrorContain: "version",
		},
		{
			name: "HTTP step missing HTTPConfig",
			step: &api.Step{
				ID:      "test-id",
				Name:    "Test",
				Version: "1.0.0",
				Type:    api.StepTypeSync,
			},
			expectedErrorContain: "http",
		},
		{
			name: "HTTP step missing endpoint",
			step: &api.Step{
				ID:      "test-id",
				Name:    "Test",
				Version: "1.0.0",
				Type:    api.StepTypeSync,
				HTTP:    &api.HTTPConfig{},
			},
			expectedErrorContain: "endpoint",
		},
		{
			name: "script step missing ScriptConfig",
			step: &api.Step{
				ID:      "test-id",
				Name:    "Test",
				Version: "1.0.0",
				Type:    api.StepTypeScript,
			},
			expectedErrorContain: "script",
		},
		{
			name: "script step missing language",
			step: &api.Step{
				ID:      "test-id",
				Name:    "Test",
				Version: "1.0.0",
				Type:    api.StepTypeScript,
				Script: &api.ScriptConfig{
					Script: "(+ 1 2)",
				},
			},
			expectedErrorContain: "language",
		},
		{
			name: "script step missing script",
			step: &api.Step{
				ID:      "test-id",
				Name:    "Test",
				Version: "1.0.0",
				Type:    api.StepTypeScript,
				Script: &api.ScriptConfig{
					Language: api.ScriptLangAle,
				},
			},
			expectedErrorContain: "script",
		},
		{
			name: "invalid step type",
			step: &api.Step{
				ID:      "test-id",
				Name:    "Test",
				Version: "1.0.0",
				Type:    "invalid",
			},
			expectedErrorContain: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := New(t)
			w.StepInvalid(tt.step, tt.expectedErrorContain)
		})
	}
}

func TestFlowStatus(t *testing.T) {
	tests := []struct {
		name           string
		flowStatus     api.FlowStatus
		expectedStatus api.FlowStatus
		shouldPass     bool
	}{
		{
			name:           "active matches active",
			flowStatus:     api.FlowActive,
			expectedStatus: api.FlowActive,
			shouldPass:     true,
		},
		{
			name:           "completed matches completed",
			flowStatus:     api.FlowCompleted,
			expectedStatus: api.FlowCompleted,
			shouldPass:     true,
		},
		{
			name:           "failed matches failed",
			flowStatus:     api.FlowFailed,
			expectedStatus: api.FlowFailed,
			shouldPass:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flow := &api.FlowState{
				Status: tt.flowStatus,
			}

			w := New(t)
			w.FlowStatus(flow, tt.expectedStatus)
		})
	}
}

func TestFlowHasState(t *testing.T) {
	tests := []struct {
		name       string
		getter     *mockGetter
		flowID     api.FlowID
		keys       []api.Name
		shouldFail bool
	}{
		{
			name: "has all required keys",
			getter: &mockGetter{
				attrs: map[api.FlowID]map[api.Name]any{
					"flow-1": {
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
			flowID: "flow-1",
			keys:   []api.Name{"key1", "key2"},
		},
		{
			name: "has single key",
			getter: &mockGetter{
				attrs: map[api.FlowID]map[api.Name]any{
					"flow-1": {
						"key1": "value1",
					},
				},
			},
			flowID: "flow-1",
			keys:   []api.Name{"key1"},
		},
		{
			name: "empty keys list",
			getter: &mockGetter{
				attrs: map[api.FlowID]map[api.Name]any{
					"flow-1": {},
				},
			},
			flowID: "flow-1",
			keys:   []api.Name{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := New(t)
			ctx := context.Background()
			w.FlowHasState(ctx, tt.getter, tt.flowID, tt.keys...)
		})
	}
}

func TestFlowStateEquals(t *testing.T) {
	tests := []struct {
		name     string
		getter   *mockGetter
		flowID   api.FlowID
		key      api.Name
		expected any
	}{
		{
			name: "string value matches",
			getter: &mockGetter{
				attrs: map[api.FlowID]map[api.Name]any{
					"flow-1": {
						"name": "test",
					},
				},
			},
			flowID:   "flow-1",
			key:      "name",
			expected: "test",
		},
		{
			name: "integer value matches",
			getter: &mockGetter{
				attrs: map[api.FlowID]map[api.Name]any{
					"flow-1": {
						"count": 42,
					},
				},
			},
			flowID:   "flow-1",
			key:      "count",
			expected: 42,
		},
		{
			name: "boolean value matches",
			getter: &mockGetter{
				attrs: map[api.FlowID]map[api.Name]any{
					"flow-1": {
						"active": true,
					},
				},
			},
			flowID:   "flow-1",
			key:      "active",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := New(t)
			ctx := context.Background()
			w.FlowStateEquals(
				ctx, tt.getter, tt.flowID, tt.key, tt.expected,
			)
		})
	}
}

func TestConfigValid(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "default config is valid",
			cfg:  config.NewDefaultConfig(),
		},
		{
			name: "custom valid config",
			cfg: &config.Config{
				APIPort:     9090,
				StepTimeout: 60000,
			},
		},
		{
			name: "minimum valid port",
			cfg: &config.Config{
				APIPort:     1,
				StepTimeout: 1000,
			},
		},
		{
			name: "maximum valid port",
			cfg: &config.Config{
				APIPort:     65535,
				StepTimeout: 1000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := New(t)
			w.ConfigValid(tt.cfg)
		})
	}
}

func TestConfigInvalid(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		contains string
	}{
		{
			name: "invalid port zero",
			cfg: &config.Config{
				APIPort:     0,
				StepTimeout: 1000,
			},
			contains: "port",
		},
		{
			name: "invalid port negative",
			cfg: &config.Config{
				APIPort:     -1,
				StepTimeout: 1000,
			},
			contains: "port",
		},
		{
			name: "invalid port too large",
			cfg: &config.Config{
				APIPort:     65536,
				StepTimeout: 1000,
			},
			contains: "port",
		},
		{
			name: "invalid step timeout zero",
			cfg: &config.Config{
				APIPort:     8080,
				StepTimeout: 0,
			},
			contains: "timeout",
		},
		{
			name: "invalid step timeout negative",
			cfg: &config.Config{
				APIPort:     8080,
				StepTimeout: -1,
			},
			contains: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := New(t)
			w.ConfigInvalid(tt.cfg, tt.contains)
		})
	}
}

func TestEventually(t *testing.T) {
	tests := []struct {
		name       string
		condition  func() bool
		timeout    time.Duration
		shouldPass bool
	}{
		{
			name: "condition passes immediately",
			condition: func() bool {
				return true
			},
			timeout:    1 * time.Second,
			shouldPass: true,
		},
		{
			name: "condition passes after retries",
			condition: func() func() bool {
				attempts := 0
				return func() bool {
					attempts++
					return attempts >= 3
				}
			}(),
			timeout:    1 * time.Second,
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := New(t)
			w.Eventually(tt.condition, tt.timeout, "condition should pass")
		})
	}
}

func TestEventuallyWithError(t *testing.T) {
	tests := []struct {
		name       string
		condition  func() error
		timeout    time.Duration
		shouldPass bool
	}{
		{
			name: "condition succeeds immediately",
			condition: func() error {
				return nil
			},
			timeout:    1 * time.Second,
			shouldPass: true,
		},
		{
			name: "condition succeeds after retries",
			condition: func() func() error {
				attempts := 0
				return func() error {
					attempts++
					if attempts >= 3 {
						return nil
					}
					return errors.New("not ready yet")
				}
			}(),
			timeout:    1 * time.Second,
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := New(t)
			w.EventuallyWithError(
				tt.condition, tt.timeout, "condition should succeed",
			)
		})
	}
}
