package helpers

import (
	"slices"
	"sync"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
)

type (
	// MockClient is a simple mock implementation of client.Client for testing
	MockClient struct {
		steps   map[api.StepID]*mockStep
		invoked []api.StepID
		mu      sync.Mutex
	}

	// mockStep is everything the mock was told about one step
	mockStep struct {
		response   api.Args
		err        error
		invoke     MockInvoke
		compensate MockCompensate
		compErr    error
		metadata   []api.Metadata
		invokedCh  chan struct{}
	}

	MockInvoke     func(*api.Step, api.Args, api.Metadata) (api.Args, error)
	MockCompensate func(step.CompensateRequest) error
)

// NewMockClient creates a mock HTTP client that allows setting responses and
// errors for specific step IDs
func NewMockClient() *MockClient {
	return &MockClient{
		steps:   map[api.StepID]*mockStep{},
		invoked: []api.StepID{},
	}
}

// Invoke records the invocation and returns the configured response or error
func (c *MockClient) Invoke(
	st *api.Step, args api.Args, md api.Metadata,
) (api.Args, error) {
	c.mu.Lock()
	c.invoked = append(c.invoked, st.ID)
	s := c.stepLocked(st.ID)
	s.metadata = append(s.metadata, md)
	if s.invokedCh != nil {
		select {
		case s.invokedCh <- struct{}{}:
		default:
		}
	}
	invoke := s.invoke
	err := s.err
	out := s.response
	c.mu.Unlock()

	if invoke != nil {
		return invoke(st, args, md)
	}
	if err != nil {
		return nil, err
	}
	if out != nil {
		return out, nil
	}
	return nil, nil
}

// Compensate records the compensate invocation and returns any configured error
func (c *MockClient) Compensate(req step.CompensateRequest) error {
	c.mu.Lock()
	s := c.stepLocked(req.Step.ID)
	compensate := s.compensate
	err := s.compErr
	c.mu.Unlock()

	if compensate != nil {
		return compensate(req)
	}
	return err
}

// SetCompensate configures a custom compensation handler for a step
func (c *MockClient) SetCompensate(sid api.StepID, fn MockCompensate) {
	c.update(sid, func(s *mockStep) { s.compensate = fn })
}

// SetCompError configures the mock to return an error on compensation
func (c *MockClient) SetCompError(sid api.StepID, err error) {
	c.update(sid, func(s *mockStep) { s.compErr = err })
}

// SetResponse configures the mock to return specific outputs for a step
func (c *MockClient) SetResponse(sid api.StepID, outputs api.Args) {
	c.update(sid, func(s *mockStep) { s.response = outputs })
}

// SetError configures the mock to return an error for a step
func (c *MockClient) SetError(sid api.StepID, err error) {
	c.update(sid, func(s *mockStep) { s.err = err })
}

// SetInvoke configures a custom invocation handler for a step
func (c *MockClient) SetInvoke(sid api.StepID, fn MockInvoke) {
	c.update(sid, func(s *mockStep) { s.invoke = fn })
}

// ClearInvoke removes a custom handler for a step
func (c *MockClient) ClearInvoke(sid api.StepID) {
	c.update(sid, func(s *mockStep) { s.invoke = nil })
}

// ClearError removes any configured error for a step
func (c *MockClient) ClearError(sid api.StepID) {
	c.update(sid, func(s *mockStep) { s.err = nil })
}

// GetInvocations returns the list of step IDs that were invoked
func (c *MockClient) GetInvocations() []api.StepID {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.invoked)
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
	s := c.stepLocked(sid)
	if s.invokedCh == nil {
		s.invokedCh = make(chan struct{}, 1)
	}
	ch := s.invokedCh
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

	entries := c.stepLocked(sid).metadata
	if len(entries) == 0 {
		return nil
	}
	return entries[len(entries)-1]
}

func (c *MockClient) update(sid api.StepID, fn func(*mockStep)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn(c.stepLocked(sid))
}

func (c *MockClient) stepLocked(sid api.StepID) *mockStep {
	s, ok := c.steps[sid]
	if !ok {
		s = &mockStep{}
		c.steps[sid] = s
	}
	return s
}

func (c *MockClient) wasInvokedLocked(sid api.StepID) bool {
	return slices.Contains(c.invoked, sid)
}
