package helpers

import (
	"slices"
	"sync"
	"time"

	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// MockClient is a simple mock implementation of client.Client for testing
	MockClient struct {
		responses    map[api.StepID]api.Args
		errors       map[api.StepID]error
		handlers     map[api.StepID]MockHandler
		compHandlers map[api.StepID]MockCompHandler
		compErrors   map[api.StepID]error
		metadata     map[api.StepID][]api.Metadata
		invokedCh    map[api.StepID]chan struct{}
		invoked      []api.StepID
		mu           sync.Mutex
	}

	MockHandler     func(*api.Step, api.Args, api.Metadata) (api.Args, error)
	MockCompHandler func(client.CompensateRequest) error
)

// NewMockClient creates a mock HTTP client that allows setting responses and
// errors for specific step IDs
func NewMockClient() *MockClient {
	return &MockClient{
		responses:    map[api.StepID]api.Args{},
		errors:       map[api.StepID]error{},
		handlers:     map[api.StepID]MockHandler{},
		compHandlers: map[api.StepID]MockCompHandler{},
		compErrors:   map[api.StepID]error{},
		invoked:      []api.StepID{},
		metadata:     map[api.StepID][]api.Metadata{},
		invokedCh:    map[api.StepID]chan struct{}{},
	}
}

// Invoke records the invocation and returns the configured response or error
func (c *MockClient) Invoke(
	st *api.Step, args api.Args, md api.Metadata,
) (api.Args, error) {
	c.mu.Lock()
	c.invoked = append(c.invoked, st.ID)
	c.metadata[st.ID] = append(c.metadata[st.ID], md)
	if ch, ok := c.invokedCh[st.ID]; ok {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	h := c.handlers[st.ID]
	err := c.errors[st.ID]
	out := c.responses[st.ID]
	c.mu.Unlock()

	if h != nil {
		return h(st, args, md)
	}
	if err != nil {
		return nil, err
	}
	if out != nil {
		return out, nil
	}
	return nil, nil
}

// InvokeCompensate records the compensate invocation and returns any
// configured error
func (c *MockClient) InvokeCompensate(req client.CompensateRequest) error {
	c.mu.Lock()
	h := c.compHandlers[req.Step.ID]
	err := c.compErrors[req.Step.ID]
	c.mu.Unlock()

	if h != nil {
		return h(req)
	}
	return err
}

// SetCompHandler configures a custom compensation handler for a step
func (c *MockClient) SetCompHandler(
	sid api.StepID, handler MockCompHandler,
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.compHandlers[sid] = handler
}

// SetCompError configures the mock to return an error on compensation
func (c *MockClient) SetCompError(sid api.StepID, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.compErrors[sid] = err
}

// SetResponse configures the mock to return specific outputs for a step
func (c *MockClient) SetResponse(sid api.StepID, outputs api.Args) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responses[sid] = outputs
}

// SetError configures the mock to return an error for a step
func (c *MockClient) SetError(sid api.StepID, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors[sid] = err
}

// SetHandler configures a custom invocation handler for a step
func (c *MockClient) SetHandler(sid api.StepID, handler MockHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[sid] = handler
}

// ClearHandler removes a custom handler for a step
func (c *MockClient) ClearHandler(sid api.StepID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.handlers, sid)
}

// ClearError removes any configured error for a step
func (c *MockClient) ClearError(sid api.StepID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.errors, sid)
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
func (c *MockClient) WasInvoked(sid api.StepID) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.wasInvokedLocked(sid)
}

// WaitForInvocation blocks until a step is invoked or the timeout expires
func (c *MockClient) WaitForInvocation(
	sid api.StepID, timeout time.Duration,
) bool {
	c.mu.Lock()
	if c.wasInvokedLocked(sid) {
		c.mu.Unlock()
		return true
	}
	ch, ok := c.invokedCh[sid]
	if !ok {
		ch = make(chan struct{}, 1)
		c.invokedCh[sid] = ch
	}
	c.mu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ch:
		return true
	case <-timer.C:
		return c.WasInvoked(sid)
	}
}

// LastMetadata returns the most recent metadata passed for a step invocation
func (c *MockClient) LastMetadata(sid api.StepID) api.Metadata {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries := c.metadata[sid]
	if len(entries) == 0 {
		return nil
	}
	return entries[len(entries)-1]
}

func (c *MockClient) wasInvokedLocked(sid api.StepID) bool {
	return slices.Contains(c.invoked, sid)
}
