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

type Client struct {
	httpClient *http.Client
	baseURL    string
}

var (
	ErrRegisterStep  = errors.New("failed to register step")
	ErrStartWorkflow = errors.New("failed to start workflow")
	ErrGetWorkflow   = errors.New("failed to get workflow")
	ErrUpdateState   = errors.New("failed to update state")
	ErrListSteps     = errors.New("failed to list steps")
)

const (
	DefaultStepPort = 8081
)

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) RegisterStep(ctx context.Context, step *api.Step) error {
	data, err := json.Marshal(step)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx, "POST", c.baseURL+"/engine/step", bytes.NewBuffer(data),
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

func (c *Client) StartWorkflow(
	ctx context.Context, flowID, goalID timebox.ID, initState api.Args,
) error {
	flow := api.CreateWorkflowRequest{
		ID:           flowID,
		Goals:        []timebox.ID{goalID},
		Init: initState,
	}
	return c.StartWorkflowWithRequest(ctx, flow)
}

func (c *Client) StartWorkflowWithRequest(
	ctx context.Context, flow api.CreateWorkflowRequest,
) error {
	data, err := json.Marshal(flow)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx, "POST", c.baseURL+"/engine/workflow", bytes.NewBuffer(data),
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
			ErrStartWorkflow, resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) GetWorkflow(
	ctx context.Context, flowID timebox.ID,
) (*api.WorkflowState, error) {
	req, err := http.NewRequestWithContext(
		ctx, "GET", c.baseURL+"/engine/workflow/"+string(flowID), nil,
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
			ErrGetWorkflow, resp.StatusCode, string(body))
	}

	var result api.WorkflowState
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) UpdateState(
	ctx context.Context, flowID timebox.ID, updates api.Args,
) error {
	data, err := json.Marshal(updates)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx, "PATCH", c.baseURL+"/engine/workflow/"+string(flowID)+"/state",
		bytes.NewBuffer(data),
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
		return fmt.Errorf("%s: status %d, body: %s",
			ErrUpdateState, resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) ListSteps(
	ctx context.Context,
) (*api.StepsListResponse, error) {
	req, err := http.NewRequestWithContext(
		ctx, "GET", c.baseURL+"/engine/step", nil,
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
