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
	Client struct {
		httpClient *http.Client
		baseURL    string
	}

	WorkflowClient struct {
		client *Client
		flowID timebox.ID
	}

	AsyncContext struct {
		flowID     timebox.ID
		stepID     timebox.ID
		webhookURL string
		httpClient *http.Client
	}
)

var (
	ErrRegisterStep  = errors.New("failed to register step")
	ErrStartWorkflow = errors.New("failed to start workflow")
	ErrGetWorkflow   = errors.New("failed to get workflow")
	ErrListSteps     = errors.New("failed to list steps")
)

const (
	DefaultStepPort = 8081

	routeSteps    = "/engine/step"
	routeWorkflow = "/engine/workflow"
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

func (c *Client) StartWorkflow(
	ctx context.Context, flowID timebox.ID, goals []timebox.ID,
	initState api.Args,
) error {
	return c.StartWorkflowWithRequest(ctx, api.CreateWorkflowRequest{
		ID:    flowID,
		Goals: goals,
		Init:  initState,
	})
}

func (c *Client) StartWorkflowWithRequest(
	ctx context.Context, flow api.CreateWorkflowRequest,
) error {
	data, err := json.Marshal(flow)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx, "POST", c.url(routeWorkflow), bytes.NewBuffer(data),
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

func (c *Client) Workflow(flowID timebox.ID) *WorkflowClient {
	return &WorkflowClient{
		client: c,
		flowID: flowID,
	}
}

func (c *Client) WorkflowFromContext(
	ctx context.Context,
) (*WorkflowClient, error) {
	meta, ok := ctx.Value(MetadataKey).(api.Metadata)
	if !ok {
		return nil, fmt.Errorf("metadata not found in context")
	}
	flowID, ok := meta["flow_id"].(string)
	if !ok || flowID == "" {
		return nil, fmt.Errorf("flow_id not found in metadata")
	}
	return c.Workflow(timebox.ID(flowID)), nil
}

func (c *Client) url(format string, args ...any) string {
	path := fmt.Sprintf(format, args...)
	return c.baseURL + path
}

func (wc *WorkflowClient) GetState(
	ctx context.Context,
) (*api.WorkflowState, error) {
	req, err := http.NewRequestWithContext(
		ctx, "GET", wc.client.url("%s/%s", routeWorkflow, wc.flowID), nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := wc.client.httpClient.Do(req)
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

func (wc *WorkflowClient) FlowID() timebox.ID {
	return wc.flowID
}

func NewAsyncContext(ctx context.Context) (*AsyncContext, error) {
	meta, ok := ctx.Value(MetadataKey).(api.Metadata)
	if !ok {
		return nil, fmt.Errorf("metadata not found in context")
	}

	flowID, ok := meta["flow_id"].(string)
	if !ok || flowID == "" {
		return nil, fmt.Errorf("flow_id not found in metadata")
	}

	stepID, ok := meta["step_id"].(string)
	if !ok || stepID == "" {
		return nil, fmt.Errorf("step_id not found in metadata")
	}

	webhookURL, ok := meta["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("webhook_url not found in metadata")
	}

	return &AsyncContext{
		flowID:     timebox.ID(flowID),
		stepID:     timebox.ID(stepID),
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (ac *AsyncContext) Success(outputs api.Args) error {
	result := api.StepResult{
		Success: true,
		Outputs: outputs,
	}
	return ac.sendWebhook(result)
}

func (ac *AsyncContext) Complete(result api.StepResult) error {
	return ac.sendWebhook(result)
}

func (ac *AsyncContext) Fail(err error) error {
	result := api.StepResult{
		Success: false,
		Error:   err.Error(),
	}
	return ac.sendWebhook(result)
}

func (ac *AsyncContext) FlowID() timebox.ID {
	return ac.flowID
}

func (ac *AsyncContext) StepID() timebox.ID {
	return ac.stepID
}

func (ac *AsyncContext) WebhookURL() string {
	return ac.webhookURL
}

func (ac *AsyncContext) sendWebhook(result api.StepResult) error {
	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	resp, err := http.Post(
		ac.webhookURL, "application/json", bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s",
			resp.StatusCode, string(body))
	}

	return nil
}
