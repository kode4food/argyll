package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kode4food/spuds/engine/pkg/api"
)

// AsyncContext provides functionality for managing asynchronous step
// execution. It embeds StepContext and adds the webhook URL for result delivery
type AsyncContext struct {
	*StepContext
	webhookURL string
}

// NewAsyncContext creates a new async context from a StepContext.
// It extracts webhook_url from the StepContext metadata
func NewAsyncContext(ctx *StepContext) (*AsyncContext, error) {
	if ctx.Metadata == nil {
		return nil, fmt.Errorf("metadata not found in step context")
	}

	webhookURL, ok := ctx.Metadata["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("webhook_url not found in metadata")
	}

	return &AsyncContext{
		StepContext: ctx,
		webhookURL:  webhookURL,
	}, nil
}

// Success marks the async step as successfully completed with the given
// outputs
func (ac *AsyncContext) Success(outputs api.Args) error {
	result := api.StepResult{
		Success: true,
		Outputs: outputs,
	}
	return ac.sendWebhook(result)
}

// Complete sends the full step result to the orchestrator via webhook
func (ac *AsyncContext) Complete(result api.StepResult) error {
	return ac.sendWebhook(result)
}

// Fail marks the async step as failed with the given error
func (ac *AsyncContext) Fail(err error) error {
	result := api.StepResult{
		Success: false,
		Error:   err.Error(),
	}
	return ac.sendWebhook(result)
}

// FlowID returns the flow ID for this async context
func (ac *AsyncContext) FlowID() string {
	return string(ac.Client.FlowID())
}

// StepID returns the step ID for this async context
func (ac *AsyncContext) StepID() string {
	return string(ac.StepContext.StepID)
}

// WebhookURL returns the webhook URL for delivering step results
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
