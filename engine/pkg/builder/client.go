package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type (
	// Client provides functionality for interacting with the orchestrator
	// API, including step registration, flow management, and state queries
	Client struct {
		httpClient *http.Client
		baseURL    string
	}

	// FlowClient provides access to a specific flow
	FlowClient struct {
		*Client
		flowID api.FlowID
	}
)

var (
	ErrRegisterStep = errors.New("failed to register step")
	ErrUpdateStep   = errors.New("failed to update step")
	ErrListSteps    = errors.New("failed to list steps")
	ErrStartFlow    = errors.New("failed to start flow")
	ErrGetFlow      = errors.New("failed to get flow")
)

const (
	DefaultStepPort = 8081

	routeSteps = "/engine/step"
	routeFlow  = "/engine/flow"
)

// NewClient creates a new orchestrator client with the specified base URL
// and timeout
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ListSteps retrieves all registered steps from the orchestrator
func (c *Client) ListSteps(
	ctx context.Context,
) (*api.StepsListResponse, error) {
	req, err := http.NewRequestWithContext(
		ctx, "GET", c.url(routeSteps), nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d, body: %s",
			ErrListSteps, resp.StatusCode, string(body))
	}

	var result api.StepsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Flow returns a client for accessing a specific flow
func (c *Client) Flow(id api.FlowID) *FlowClient {
	return &FlowClient{
		Client: c,
		flowID: id,
	}
}

func (c *Client) url(format string, args ...any) string {
	path := fmt.Sprintf(format, args...)
	return c.baseURL + path
}

func (c *Client) registerStep(ctx context.Context, step *api.Step) error {
	data, err := json.Marshal(step)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx, "POST", c.url(routeSteps), bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d, body: %s",
			ErrRegisterStep, resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) updateStep(ctx context.Context, step *api.Step) error {
	data, err := json.Marshal(step)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx, "PUT", c.url("%s/%s", routeSteps, step.ID), bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d, body: %s",
			ErrUpdateStep, resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) startFlow(
	ctx context.Context, req *api.CreateFlowRequest,
) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, "POST", c.url(routeFlow), bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d, body: %s",
			ErrStartFlow, resp.StatusCode, string(body))
	}

	return nil
}

// GetState retrieves the current state of the flow
func (c *FlowClient) GetState(ctx context.Context) (*api.FlowState, error) {
	httpReq, err := http.NewRequestWithContext(
		ctx, "GET", c.url("%s/%s", routeFlow, c.flowID), nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d, body: %s",
			ErrGetFlow, resp.StatusCode, string(body))
	}

	var result api.FlowState
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// FlowID returns the flow ID for this client
func (c *FlowClient) FlowID() api.FlowID {
	return c.flowID
}
