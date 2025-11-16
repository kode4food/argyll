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

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type (
	// Client provides functionality for interacting with the workflow engine
	// API, including step registration, workflow management, and state queries
	Client struct {
		httpClient *http.Client
		baseURL    string
	}

	// WorkflowClient provides access to a specific workflow
	WorkflowClient struct {
		client *Client
		flowID timebox.ID
	}
)

var (
	ErrRegisterStep  = errors.New("failed to register step")
	ErrListSteps     = errors.New("failed to list steps")
	ErrStartWorkflow = errors.New("failed to start workflow")
	ErrGetWorkflow   = errors.New("failed to get workflow")
)

const (
	DefaultStepPort = 8081

	routeSteps    = "/engine/step"
	routeWorkflow = "/engine/workflow"
)

// NewClient creates a new workflow engine client with the specified base URL
// and timeout
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ListSteps retrieves all registered steps from the workflow engine
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
		return nil, fmt.Errorf("%s: status %d, body: %s",
			ErrListSteps, resp.StatusCode, string(body))
	}

	var result api.StepsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Workflow returns a client for accessing a specific workflow
func (c *Client) Workflow(flowID timebox.ID) *WorkflowClient {
	return &WorkflowClient{
		client: c,
		flowID: flowID,
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
		return fmt.Errorf("%s: status %d, body: %s",
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
		return fmt.Errorf("failed to update step: status %d, body: %s",
			resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) startWorkflow(
	ctx context.Context, req *api.CreateWorkflowRequest,
) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, "POST", c.url(routeWorkflow), bytes.NewBuffer(data),
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
		return fmt.Errorf("%s: status %d, body: %s",
			ErrStartWorkflow, resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) getWorkflowState(
	ctx context.Context, flowID timebox.ID,
) (*api.WorkflowState, error) {
	httpReq, err := http.NewRequestWithContext(
		ctx, "GET", c.url("%s/%s", routeWorkflow, flowID), nil,
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
		return nil, fmt.Errorf("%s: status %d, body: %s",
			ErrGetWorkflow, resp.StatusCode, string(body))
	}

	var result api.WorkflowState
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetState retrieves the current state of the workflow
func (wc *WorkflowClient) GetState(
	ctx context.Context,
) (*api.WorkflowState, error) {
	return wc.client.getWorkflowState(ctx, wc.flowID)
}

// FlowID returns the workflow ID for this client
func (wc *WorkflowClient) FlowID() timebox.ID {
	return wc.flowID
}
