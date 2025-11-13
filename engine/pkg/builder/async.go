package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type AsyncContext struct {
	client     *Client
	flowID     timebox.ID
	stepID     timebox.ID
	webhookURL string
	httpClient *http.Client
}

func (c *Client) NewAsyncContext(ctx context.Context) (*AsyncContext, error) {
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
		client:     c,
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

func (ac *AsyncContext) Workflow() *WorkflowClient {
	return ac.client.Workflow(ac.flowID)
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
