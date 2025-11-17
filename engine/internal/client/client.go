package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type (
	// Client defines the interface for invoking step handlers
	Client interface {
		Invoke(
			context.Context, *api.Step, api.Args, api.Metadata,
		) (api.Args, error)
	}

	// HTTPClient implements Client using HTTP requests
	HTTPClient struct {
		httpClient *http.Client
		timeout    time.Duration
	}
)

var (
	ErrStepUnsuccessful = errors.New("step returned success=false")
	ErrHTTPError        = errors.New("step returned HTTP error")
	ErrNoHTTPConfig     = errors.New("step has no HTTP configuration")
)

var _ Client = (*HTTPClient)(nil)

// NewHTTPClient creates a new HTTP client with the specified request timeout
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Invoke sends an HTTP POST request to the step's endpoint with the provided
// arguments and metadata, returning the step's output arguments or an error
func (c *HTTPClient) Invoke(
	ctx context.Context, step *api.Step, args api.Args, meta api.Metadata,
) (api.Args, error) {
	if step.HTTP == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoHTTPConfig, step.ID)
	}

	httpReq, err := c.buildRequest(ctx, step, args, meta)
	if err != nil {
		return nil, err
	}

	respBody, err := c.sendRequest(step, httpReq)
	if err != nil {
		return nil, err
	}

	return c.parseResponse(step, respBody)
}

func (c *HTTPClient) buildRequest(
	ctx context.Context, step *api.Step, args api.Args, meta api.Metadata,
) (*http.Request, error) {
	body, err := json.Marshal(api.StepRequest{
		Arguments: args,
		Metadata:  meta,
	})
	if err != nil {
		slog.Error("Failed to marshal step request",
			slog.Any("step_id", step.ID),
			slog.Any("error", err))
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, "POST", step.HTTP.Endpoint, bytes.NewBuffer(body),
	)
	if err != nil {
		slog.Error("Failed to create HTTP request",
			slog.Any("step_id", step.ID),
			slog.Any("error", err))
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "Spuds-Engine/1.0")

	return httpReq, nil
}

func (c *HTTPClient) sendRequest(
	step *api.Step, httpReq *http.Request,
) ([]byte, error) {
	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	dur := time.Since(start)

	if err != nil {
		slog.Error("HTTP request failed",
			slog.Any("step_id", step.ID),
			slog.Duration("duration", dur),
			slog.Any("error", err))
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read response body",
			slog.Any("step_id", step.ID),
			slog.Any("error", err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("HTTP error",
			slog.Any("step_id", step.ID),
			slog.Int("status_code", resp.StatusCode),
			slog.String("response_body", string(respBody)))
		return nil, fmt.Errorf("%s: HTTP %d", ErrHTTPError, resp.StatusCode)
	}

	return respBody, nil
}

func (c *HTTPClient) parseResponse(
	step *api.Step, respBody []byte,
) (api.Args, error) {
	var response api.StepResult
	if err := json.Unmarshal(respBody, &response); err != nil {
		slog.Error("Failed to unmarshal response",
			slog.Any("step_id", step.ID),
			slog.Any("error", err))
		return nil, err
	}

	if !response.Success {
		if response.Error == "" {
			slog.Error("Step unsuccessful",
				slog.Any("step_id", step.ID))
			return nil, ErrStepUnsuccessful
		}
		slog.Error("Step failed",
			slog.Any("step_id", step.ID),
			slog.String("error", response.Error))
		return nil, fmt.Errorf("%w: %s", ErrStepUnsuccessful, response.Error)
	}

	return response.Outputs, nil
}
