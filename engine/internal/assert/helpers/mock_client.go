package helpers

import (
	"slices"
	"sync"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// MockClient is a simple mock implementation of client.Client for testing
type MockClient struct {
	responses map[api.StepID]api.Args
	errors    map[api.StepID]error
	handlers  map[api.StepID]func(*api.Step, api.Args, api.Metadata) (
		api.Args, error,
	)
	invoked   []api.StepID
	metadata  map[api.StepID][]api.Metadata
	invokedCh map[api.StepID]chan struct{}
	mu        sync.Mutex
}

// NewMockClient creates a mock HTTP client that allows setting responses and
// errors for specific step IDs
func NewMockClient() *MockClient {
	return &MockClient{
		responses: map[api.StepID]api.Args{},
		errors:    map[api.StepID]error{},
		handlers: map[api.StepID]func(*api.Step, api.Args, api.Metadata) (
			api.Args, error,
		){},
		invoked:   []api.StepID{},
		metadata:  map[api.StepID][]api.Metadata{},
		invokedCh: map[api.StepID]chan struct{}{},
	}
}

// Invoke records the invocation and returns the configured response or error
func (c *MockClient) Invoke(
	step *api.Step, args api.Args, md api.Metadata,
) (api.Args, error) {
	c.mu.Lock()
	c.invoked = append(c.invoked, step.ID)
	c.metadata[step.ID] = append(c.metadata[step.ID], md)
	if ch, ok := c.invokedCh[step.ID]; ok {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	h := c.handlers[step.ID]
	err := c.errors[step.ID]
	out := c.responses[step.ID]
	c.mu.Unlock()

	if h != nil {
		return h(step, args, md)
	}
	if err != nil {
		return nil, err
	}
	if out != nil {
		return out, nil
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

// SetHandler configures a custom invocation handler for a step
func (c *MockClient) SetHandler(
	stepID api.StepID,
	handler func(*api.Step, api.Args, api.Metadata) (api.Args, error),
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[stepID] = handler
}

// ClearHandler removes a custom handler for a step
func (c *MockClient) ClearHandler(stepID api.StepID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.handlers, stepID)
}

// ClearError removes any configured error for a step
func (c *MockClient) ClearError(stepID api.StepID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.errors, stepID)
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
	return c.wasInvokedLocked(stepID)
}

// WaitForInvocation blocks until a step is invoked or the timeout expires
func (c *MockClient) WaitForInvocation(
	stepID api.StepID, timeout time.Duration,
) bool {
	c.mu.Lock()
	if c.wasInvokedLocked(stepID) {
		c.mu.Unlock()
		return true
	}
	ch, ok := c.invokedCh[stepID]
	if !ok {
		ch = make(chan struct{}, 1)
		c.invokedCh[stepID] = ch
	}
	c.mu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ch:
		return true
	case <-timer.C:
		return c.WasInvoked(stepID)
	}
}

// LastMetadata returns the most recent metadata passed for a step invocation
func (c *MockClient) LastMetadata(stepID api.StepID) api.Metadata {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries := c.metadata[stepID]
	if len(entries) == 0 {
		return nil
	}
	return entries[len(entries)-1]
}

func (c *MockClient) wasInvokedLocked(stepID api.StepID) bool {
	return slices.Contains(c.invoked, stepID)
}
