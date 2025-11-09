package helpers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/internal/config"
	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/pkg/api"
)

type (
	// TestEngineEnv holds all the components needed for engine testing
	TestEngineEnv struct {
		Engine     *engine.Engine
		Redis      *miniredis.Miniredis
		MockClient *MockClient
		Config     *config.Config
		EventHub   *timebox.EventHub
		Cleanup    func()
	}

	// MockClient is a simple mock implementation of client.Client for testing
	MockClient struct {
		responses map[timebox.ID]api.Args
		errors    map[timebox.ID]error
		invoked   []timebox.ID
		mu        sync.Mutex
	}
)

// NewTestStep creates a basic step for testing
func NewTestStep() *api.Step {
	return &api.Step{
		ID:   timebox.ID("test-step-" + uuid.New().String()[:8]),
		Name: "Test Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080/transform",
			Timeout:  30 * api.Second,
		},
		Attributes: map[api.Name]*api.AttributeSpec{
			"input": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"optional": {
				Role: api.RoleOptional,
				Type: api.TypeString,
			},
			"output": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		},
		Version: "1.0.0",
	}
}

// NewTestStepWithArgs creates a step with specific arguments
func NewTestStepWithArgs(required []api.Name, optional []api.Name) *api.Step {
	step := NewTestStep()

	step.Attributes = map[api.Name]*api.AttributeSpec{}
	for _, arg := range required {
		step.Attributes[arg] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}
	}

	for _, arg := range optional {
		step.Attributes[arg] = &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
		}
	}

	return step
}

// NewSimpleStep creates a minimal HTTP step with specific ID
func NewSimpleStep(id timebox.ID) *api.Step {
	return &api.Step{
		ID:         id,
		Name:       "Test Step",
		Type:       api.StepTypeSync,
		Version:    "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080",
		},
	}
}

// NewStepWithOutputs creates an HTTP step with specific outputs
func NewStepWithOutputs(id timebox.ID, outputs ...api.Name) *api.Step {
	step := NewSimpleStep(id)
	if step.Attributes == nil {
		step.Attributes = map[api.Name]*api.AttributeSpec{}
	}
	for _, name := range outputs {
		step.Attributes[name] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}
	}
	return step
}

// NewScriptStep creates a script-based step
func NewScriptStep(
	id timebox.ID, language, script string, outputs ...api.Name,
) *api.Step {
	step := &api.Step{
		ID:      id,
		Name:    "Script Step",
		Type:    api.StepTypeScript,
		Version: "1.0.0",
		Script: &api.ScriptConfig{
			Language: language,
			Script:   script,
		},
		Attributes: map[api.Name]*api.AttributeSpec{},
	}
	for _, name := range outputs {
		step.Attributes[name] = &api.AttributeSpec{
			Role: api.RoleOutput,
		}
	}
	return step
}

// NewStepWithPredicate creates an HTTP step with a predicate
func NewStepWithPredicate(
	id timebox.ID, predicateLang, predicateScript string, outputs ...api.Name,
) *api.Step {
	step := NewSimpleStep(id)
	step.Predicate = &api.ScriptConfig{
		Language: predicateLang,
		Script:   predicateScript,
	}
	if step.Attributes == nil {
		step.Attributes = map[api.Name]*api.AttributeSpec{}
	}
	for _, name := range outputs {
		step.Attributes[name] = &api.AttributeSpec{
			Role: api.RoleOutput,
		}
	}
	return step
}

// NewTestConfig creates a basic configuration for testing
func NewTestConfig() *config.Config {
	cfg := config.NewDefaultConfig()
	cfg.LogLevel = "debug"
	return cfg
}

// NewMockClient creates a new mock HTTP client for testing
func NewMockClient() *MockClient {
	return &MockClient{
		responses: map[timebox.ID]api.Args{},
		errors:    map[timebox.ID]error{},
		invoked:   []timebox.ID{},
	}
}

// NewTestEngine creates a test engine with miniredis backend
func NewTestEngine(t *testing.T) *TestEngineEnv {
	t.Helper()

	server, err := miniredis.Run()
	require.NoError(t, err)

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
	})
	require.NoError(t, err)

	engineConfig := config.NewDefaultConfig().EngineStore
	engineConfig.Addr = server.Addr()
	engineConfig.Prefix = "test-engine"

	engineStore, err := tb.NewStore(engineConfig)
	require.NoError(t, err)

	workflowConfig := config.NewDefaultConfig().WorkflowStore
	workflowConfig.Addr = server.Addr()
	workflowConfig.Prefix = "test-workflow"

	workflowStore, err := tb.NewStore(workflowConfig)
	require.NoError(t, err)

	mockCli := NewMockClient()

	cfg := &config.Config{
		APIPort:            8080,
		APIHost:            "localhost",
		WebhookBaseURL:     "http://localhost:8080",
		StepTimeout:        5 * api.Second,
		MaxWorkflows:       100,
		WorkflowCacheSize:  100,
		ShutdownTimeout:    2 * time.Second,
		RetryCheckInterval: 100 * time.Millisecond,
		WorkConfig: api.WorkConfig{
			MaxRetries:   3,
			BackoffMs:    1000,
			MaxBackoffMs: 60000,
			BackoffType:  api.BackoffTypeExponential,
		},
	}

	eng := engine.New(engineStore, workflowStore, mockCli, tb.GetHub(), cfg)

	cleanup := func() {
		_ = eng.Stop()
		_ = tb.Close()
		server.Close()
	}

	hub := tb.GetHub()
	return &TestEngineEnv{
		Engine:     eng,
		Redis:      server,
		MockClient: mockCli,
		Config:     cfg,
		EventHub:   &hub,
		Cleanup:    cleanup,
	}
}

func (m *MockClient) Invoke(
	_ context.Context, step *api.Step, _ api.Args, _ api.Metadata,
) (api.Args, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.invoked = append(m.invoked, step.ID)

	if err, ok := m.errors[step.ID]; ok {
		return nil, err
	}

	if outputs, ok := m.responses[step.ID]; ok {
		return outputs, nil
	}

	return api.Args{}, nil
}

func (m *MockClient) SetResponse(stepID timebox.ID, outputs api.Args) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[stepID] = outputs
}

func (m *MockClient) SetError(stepID timebox.ID, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[stepID] = err
}

func (m *MockClient) GetInvocations() []timebox.ID {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]timebox.ID, len(m.invoked))
	copy(result, m.invoked)
	return result
}

func (m *MockClient) WasInvoked(stepID timebox.ID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.invoked {
		if id == stepID {
			return true
		}
	}
	return false
}
