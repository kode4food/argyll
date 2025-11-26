package helpers

import (
	"context"
	"encoding/json"
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
		responses map[api.StepID]api.Args
		errors    map[api.StepID]error
		invoked   []api.StepID
		mu        sync.Mutex
	}
)

// NewTestStep creates a basic HTTP step for testing with required, optional,
// and output attributes
func NewTestStep() *api.Step {
	return &api.Step{
		ID:   api.StepID("test-step-" + uuid.New().String()[:8]),
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

// NewTestStepWithArgs creates an HTTP step with the specified required and
// optional input arguments
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

// NewSimpleStep creates a minimal HTTP step with the specified ID and no
// attributes
func NewSimpleStep(id api.StepID) *api.Step {
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

// NewStepWithOutputs creates an HTTP step that produces the specified output
// attributes
func NewStepWithOutputs(id api.StepID, outputs ...api.Name) *api.Step {
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

// NewScriptStep creates a script-based step with the specified language, code,
// and output attributes
func NewScriptStep(
	id api.StepID, language, script string, outputs ...api.Name,
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

// NewStepWithPredicate creates an HTTP step with a predicate script that
// determines whether the step should execute
func NewStepWithPredicate(
	id api.StepID, lang, script string, outputs ...api.Name,
) *api.Step {
	step := NewSimpleStep(id)
	step.Predicate = &api.ScriptConfig{
		Language: lang,
		Script:   script,
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

// NewTestConfig creates a default configuration with debug logging enabled for
// testing
func NewTestConfig() *config.Config {
	cfg := config.NewDefaultConfig()
	cfg.LogLevel = "debug"
	return cfg
}

// NewMockClient creates a mock HTTP client that allows setting responses and
// errors for specific step IDs
func NewMockClient() *MockClient {
	return &MockClient{
		responses: map[api.StepID]api.Args{},
		errors:    map[api.StepID]error{},
		invoked:   []api.StepID{},
	}
}

// NewTestEngine creates a fully configured test engine environment with an
// in-memory Redis backend and mock HTTP client
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

	flowConfig := config.NewDefaultConfig().FlowStore
	flowConfig.Addr = server.Addr()
	flowConfig.Prefix = "test-flow"

	flowStore, err := tb.NewStore(flowConfig)
	require.NoError(t, err)

	mockCli := NewMockClient()

	cfg := &config.Config{
		APIPort:            8080,
		APIHost:            "localhost",
		WebhookBaseURL:     "http://localhost:8080",
		StepTimeout:        5 * api.Second,
		FlowCacheSize:      100,
		ShutdownTimeout:    2 * time.Second,
		RetryCheckInterval: 100 * time.Millisecond,
		WorkConfig: api.WorkConfig{
			MaxRetries:   3,
			BackoffMs:    1000,
			MaxBackoffMs: 60000,
			BackoffType:  api.BackoffTypeExponential,
		},
	}

	eng := engine.New(engineStore, flowStore, mockCli, tb.GetHub(), cfg)

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

// Invoke records the invocation and returns the configured response or error
func (c *MockClient) Invoke(
	_ context.Context, step *api.Step, _ api.Args, _ api.Metadata,
) (api.Args, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.invoked = append(c.invoked, step.ID)

	if err, ok := c.errors[step.ID]; ok {
		return nil, err
	}

	if outputs, ok := c.responses[step.ID]; ok {
		return outputs, nil
	}

	return nil, nil
}

// SetResponse configures the mock to return specific outputs for a step
func (c *MockClient) SetResponse(stepID api.StepID, outputs api.Args) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responses[stepID] = outputs
}

// SetError configures the mock to return an error for a step
func (c *MockClient) SetError(stepID api.StepID, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors[stepID] = err
}

// GetInvocations returns the list of step IDs that were invoked
func (c *MockClient) GetInvocations() []api.StepID {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]api.StepID, len(c.invoked))
	copy(result, c.invoked)
	return result
}

// WasInvoked returns whether a specific step was invoked
func (c *MockClient) WasInvoked(stepID api.StepID) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range c.invoked {
		if id == stepID {
			return true
		}
	}
	return false
}

// WaitForFlowStatus waits for a flow to reach a terminal status (completed or
// failed) by subscribing to events from the event hub. Returns the final flow
// state
func (env *TestEngineEnv) WaitForFlowStatus(
	t *testing.T, ctx context.Context, flowID api.FlowID, timeout time.Duration,
) *api.FlowState {
	t.Helper()

	consumer := (*env.EventHub).NewConsumer()
	defer consumer.Close()

	deadline := time.After(timeout)

	for {
		select {
		case event := <-consumer.Receive():
			if event == nil {
				continue
			}

			eventType := api.EventType(event.Type)

			// Check for flow completed event
			if eventType == api.EventTypeFlowCompleted {
				var fc api.FlowCompletedEvent
				if err := json.Unmarshal(event.Data, &fc); err == nil &&
					fc.FlowID == flowID {
					flow, err := env.Engine.GetFlowState(ctx, flowID)
					require.NoError(t, err)
					return flow
				}
			}

			// Check for flow failed event
			if eventType == api.EventTypeFlowFailed {
				var ff api.FlowFailedEvent
				if err := json.Unmarshal(event.Data, &ff); err == nil &&
					ff.FlowID == flowID {
					flow, err := env.Engine.GetFlowState(ctx, flowID)
					require.NoError(t, err)
					return flow
				}
			}

		case <-deadline:
			// Timeout - get current state and return it
			flow, err := env.Engine.GetFlowState(ctx, flowID)
			require.NoError(t, err)
			return flow

		case <-ctx.Done():
			t.Fatal("context cancelled while waiting for flow status")
			return nil
		}
	}
}

// WaitForStepStatus waits for a specific step to reach a terminal status
func (env *TestEngineEnv) WaitForStepStatus(
	t *testing.T, ctx context.Context, flowID api.FlowID, stepID api.StepID,
	timeout time.Duration,
) *api.ExecutionState {
	t.Helper()

	consumer := (*env.EventHub).NewConsumer()
	defer consumer.Close()

	deadline := time.After(timeout)

	for {
		select {
		case event := <-consumer.Receive():
			if event == nil {
				continue
			}

			eventType := api.EventType(event.Type)

			// Check for step completed
			if eventType == api.EventTypeStepCompleted {
				var sc api.StepCompletedEvent
				if err := json.Unmarshal(event.Data, &sc); err == nil {
					if sc.FlowID == flowID && sc.StepID == stepID {
						flow, err := env.Engine.GetFlowState(ctx, flowID)
						require.NoError(t, err)
						return flow.Executions[stepID]
					}
				}
			}

			// Check for step failed
			if eventType == api.EventTypeStepFailed {
				var sf api.StepFailedEvent
				if err := json.Unmarshal(event.Data, &sf); err == nil {
					if sf.FlowID == flowID && sf.StepID == stepID {
						flow, err := env.Engine.GetFlowState(ctx, flowID)
						require.NoError(t, err)
						return flow.Executions[stepID]
					}
				}
			}

			// Check for step skipped
			if eventType == api.EventTypeStepSkipped {
				var ss api.StepSkippedEvent
				if err := json.Unmarshal(event.Data, &ss); err == nil {
					if ss.FlowID == flowID && ss.StepID == stepID {
						flow, err := env.Engine.GetFlowState(ctx, flowID)
						require.NoError(t, err)
						return flow.Executions[stepID]
					}
				}
			}

		case <-deadline:
			flow, err := env.Engine.GetFlowState(ctx, flowID)
			require.NoError(t, err)
			return flow.Executions[stepID]

		case <-ctx.Done():
			t.Fatal("context cancelled while waiting for step status")
			return nil
		}
	}
}
