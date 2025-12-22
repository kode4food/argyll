package helpers

import (
	"context"
	"sync"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// MockClient is a simple mock implementation of client.Client for testing
type MockClient struct {
	responses map[api.StepID]api.Args
	errors    map[api.StepID]error
	invoked   []api.StepID
	metadata  map[api.StepID][]api.Metadata
	mu        sync.Mutex
}

// NewMockClient creates a mock HTTP client that allows setting responses and
// errors for specific step IDs
func NewMockClient() *MockClient {
	return &MockClient{
		responses: map[api.StepID]api.Args{},
		errors:    map[api.StepID]error{},
		invoked:   []api.StepID{},
		metadata:  map[api.StepID][]api.Metadata{},
	}
}

// Invoke records the invocation and returns the configured response or error
func (c *MockClient) Invoke(
	_ context.Context, step *api.Step, _ api.Args, md api.Metadata,
) (api.Args, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.invoked = append(c.invoked, step.ID)
	c.metadata[step.ID] = append(c.metadata[step.ID], md)

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
	for _, id := range c.invoked {
		if id == stepID {
			return true
		}
	}
	return false
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
