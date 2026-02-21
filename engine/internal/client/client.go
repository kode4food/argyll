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

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// Client defines the interface for invoking step handlers
	Client interface {
		Invoke(*api.Step, api.Args, api.Metadata) (api.Args, error)
	}

	// HTTPClient implements Client using HTTP requests
	HTTPClient struct {
		httpClient *http.Client
		timeout    time.Duration
	}
)

var (
	ErrStepUnsuccessful = errors.New("step unsuccessful")
	ErrHTTPError        = errors.New("step returned HTTP error")
	ErrNoHTTPConfig     = errors.New("step has no HTTP configuration")
)

var _ Client = (*HTTPClient)(nil)

// NewHTTPClient creates a new HTTP client with the specified request timeout
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		httpClient: &http.Client{},
		timeout:    timeout,
	}
}

// Invoke sends an HTTP POST request to the step's endpoint with the provided
// arguments and metadata, returning the step's output arguments or an error
func (c *HTTPClient) Invoke(
	step *api.Step, args api.Args, meta api.Metadata,
) (api.Args, error) {
	if step.HTTP == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoHTTPConfig, step.ID)
	}

	httpReq, err := c.buildRequest(step, args, meta)
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
	step *api.Step, args api.Args, meta api.Metadata,
) (*http.Request, error) {
	body, err := json.Marshal(api.StepRequest{
		Arguments: args,
		Metadata:  meta,
	})
	if err != nil {
		slog.Error("Failed to marshal step request",
			log.StepID(step.ID),
			log.Error(err))
		return nil, err
	}

	httpReq, err := http.NewRequest(
		"POST", step.HTTP.Endpoint, bytes.NewBuffer(body),
	)
	if err != nil {
		slog.Error("Failed to create HTTP request",
			log.StepID(step.ID),
			log.Error(err))
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "Argyll-Engine/1.0")

	return httpReq, nil
}

func (c *HTTPClient) sendRequest(
	step *api.Step, httpReq *http.Request,
) ([]byte, error) {
	timeout := c.requestTimeout(step)
	ctx, cancel := context.WithTimeout(httpReq.Context(), timeout)
	defer cancel()

	req := httpReq.Clone(ctx)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	dur := time.Since(start)

	if err != nil {
		slog.Error("HTTP request failed",
			log.StepID(step.ID),
			slog.Int("duration_ms", int(dur.Milliseconds())),
			log.Error(err))
		return nil, fmt.Errorf("%w: %w", api.ErrWorkNotCompleted, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read response body",
			log.StepID(step.ID),
			log.Error(err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("HTTP error",
			log.StepID(step.ID),
			log.Error(fmt.Errorf("status %d", resp.StatusCode)),
			slog.Int("status_code", resp.StatusCode),
			slog.String("response_body", string(respBody)))

		// 4xx errors are permanent failures
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return nil, fmt.Errorf("%w: status %d",
				ErrHTTPError, resp.StatusCode)
		}

		// 5xx errors are transient
		return nil, fmt.Errorf("%w: %w: status %d",
			api.ErrWorkNotCompleted, ErrHTTPError, resp.StatusCode)
	}

	return respBody, nil
}

func (c *HTTPClient) requestTimeout(step *api.Step) time.Duration {
	if step != nil && step.HTTP != nil && step.HTTP.Timeout > 0 {
		return time.Duration(step.HTTP.Timeout) * time.Millisecond
	}
	return c.timeout
}

func (c *HTTPClient) parseResponse(
	step *api.Step, respBody []byte,
) (api.Args, error) {
	var response api.StepResult
	if err := json.Unmarshal(respBody, &response); err != nil {
		slog.Error("Failed to unmarshal response",
			log.StepID(step.ID),
			log.Error(err))
		return nil, err
	}

	if !response.Success {
		if response.Error == "" {
			slog.Error("Step unsuccessful",
				log.StepID(step.ID),
				log.Error(ErrStepUnsuccessful))
			return nil, ErrStepUnsuccessful
		}
		slog.Error("Step failed",
			log.StepID(step.ID),
			log.ErrorString(response.Error))
		return nil, fmt.Errorf("%w: %s", ErrStepUnsuccessful, response.Error)
	}

	return response.Outputs, nil
}
