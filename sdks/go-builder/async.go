package builder

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// AsyncContext provides functionality for managing asynchronous step
// execution. It embeds StepContext and adds the webhook URL for result
// delivery
type AsyncContext struct {
	*StepContext
	webhookURL string
}

var (
	ErrMetadataNotFound   = errors.New("metadata not found in step context")
	ErrWebhookURLNotFound = errors.New("webhook_url not found in metadata")
	ErrWebhookError       = errors.New("webhook returned error status")
)

// NewAsyncContext creates a new async context from a StepContext. It extracts
// webhook_url from the StepContext metadata
func NewAsyncContext(ctx *StepContext) (*AsyncContext, error) {
	if ctx.Metadata == nil {
		return nil, ErrMetadataNotFound
	}

	webhookURL, ok := ctx.Metadata[api.MetaWebhookURL].(string)
	if !ok || webhookURL == "" {
		return nil, ErrWebhookURLNotFound
	}

	return &AsyncContext{
		StepContext: ctx,
		webhookURL:  webhookURL,
	}, nil
}

// Success marks the async step as successfully completed with the given
// outputs
func (c *AsyncContext) Success(outputs api.Args) error {
	result := api.StepResult{
		Success: true,
		Outputs: outputs,
	}
	return c.sendWebhook(result)
}

// Complete sends the full step result to the orchestrator via webhook
func (c *AsyncContext) Complete(result api.StepResult) error {
	return c.sendWebhook(result)
}

// Fail marks the async step as failed with the given error
func (c *AsyncContext) Fail(err error) error {
	result := api.StepResult{
		Success: false,
		Error:   err.Error(),
	}
	return c.sendWebhook(result)
}

// FlowID returns the flow ID for this async context
func (c *AsyncContext) FlowID() string {
	return string(c.Client.FlowID())
}

// StepID returns the step ID for this async context
func (c *AsyncContext) StepID() string {
	return string(c.StepContext.StepID)
}

// WebhookURL returns the webhook URL for delivering step results
func (c *AsyncContext) WebhookURL() string {
	return c.webhookURL
}

func (c *AsyncContext) sendWebhook(result api.StepResult) error {
	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	resp, err := http.Post(
		c.webhookURL, "application/json", bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s",
			ErrWebhookError, resp.StatusCode, string(body))
	}

	return nil
}
