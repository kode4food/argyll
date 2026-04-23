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

// AsyncContext provides functionality to manage asynchronous step execution
// and embeds StepContext with the webhook URL for result delivery
type AsyncContext struct {
	*StepContext
	webhookURL string
}

var (
	ErrMetadataNotFound   = errors.New("metadata not found in step context")
	ErrWebhookURLNotFound = errors.New("webhook_url not found in metadata")
	ErrWebhookError       = errors.New("webhook returned error status")
)

// NewAsyncContext creates a new async context from a StepContext and extracts
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

// Success marks an async step as successfully completed with the given outputs
func (c *AsyncContext) Success(outputs api.Args) error {
	return c.sendWebhook(api.JSONContentType, outputs)
}

// Complete sends output arguments to the orchestrator via webhook
func (c *AsyncContext) Complete(outputs api.Args) error {
	return c.Success(outputs)
}

// Fail marks the async step as failed with the given error
func (c *AsyncContext) Fail(err error) error {
	problem := api.NewProblem(http.StatusUnprocessableEntity, err.Error())
	return c.sendWebhook(api.ProblemJSONContentType, problem)
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

func (c *AsyncContext) sendWebhook(contentType string, body any) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook body: %w", err)
	}

	resp, err := http.Post(
		c.webhookURL, contentType, bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", ErrWebhookError,
			resp.StatusCode, string(body))
	}

	return nil
}
