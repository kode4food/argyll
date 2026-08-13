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
	"net/url"
	"regexp"
	"time"

	argyll "github.com/kode4food/argyll/engine"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// Client defines the interface for invoking step handlers
	Client interface {
		Invoke(*api.Step, api.Args, api.Metadata) (api.Args, error)
		InvokeCompensate(CompensateRequest) error
	}

	// CompensateRequest carries the work item being reversed, holding the
	// inputs it ran with and the outputs it produced
	CompensateRequest struct {
		Step     *api.Step
		Inputs   api.Args
		Outputs  api.Args
		Metadata api.Metadata
	}

	// HTTPClient implements Client using HTTP requests
	HTTPClient struct {
		httpClient *http.Client
		timeout    time.Duration
	}

	compensateBody struct {
		Input  api.Args `json:"input"`
		Output api.Args `json:"output"`
	}
)

const UserAgent = "Argyll-Engine/" + argyll.Version

var (
	ErrHTTPError          = errors.New("step returned HTTP error")
	ErrNoHTTPConfig       = errors.New("step has no HTTP configuration")
	ErrMissingEndpointArg = errors.New("missing endpoint argument")
	ErrInvalidOutputJSON  = errors.New("invalid output JSON")
)

var endpointParamPattern = regexp.MustCompile(`\{([^{}]+)\}`)

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

	httpReq, err := buildRequest(step, args, meta)
	if err != nil {
		return nil, err
	}

	respBody, err := c.sendRequest(
		step, c.requestTimeout(step.HTTP.Invoke.Timeout), httpReq,
	)
	if err != nil {
		return nil, err
	}

	return parseResponse(step, respBody)
}

// InvokeCompensate sends a compensation request to the step's compensate
// endpoint with the original inputs and the successful outputs
func (c *HTTPClient) InvokeCompensate(req CompensateRequest) error {
	step := req.Step
	if step.HTTP == nil || step.HTTP.Compensate == nil {
		return fmt.Errorf("%w: %s", ErrNoHTTPConfig, step.ID)
	}
	action := step.HTTP.Compensate

	merged := req.Inputs.Apply(req.Outputs)
	endpoint, err := resolveEndpoint(action.Endpoint, merged)
	if err != nil {
		slog.Error("Failed to resolve compensate endpoint",
			log.StepID(step.ID),
			log.Error(err))
		return err
	}

	method := action.DefaultedMethod()
	bodyReader, err := compensateRequestBody(method, req)
	if err != nil {
		slog.Error("Failed to marshal compensate request",
			log.StepID(step.ID),
			log.Error(err))
		return err
	}

	httpReq, err := http.NewRequest(method, endpoint, bodyReader)
	if err != nil {
		slog.Error("Failed to create compensate request",
			log.StepID(step.ID),
			log.Error(err))
		return err
	}

	httpReq.Header.Set("Accept", api.JSONContentType)
	httpReq.Header.Set("Content-Type", api.JSONContentType)
	httpReq.Header.Set("User-Agent", UserAgent)
	api.SetMetadataHeaders(httpReq.Header, req.Metadata)

	_, err = c.sendRequest(
		step, c.requestTimeout(step.HTTP.CompensateTimeout()), httpReq,
	)
	return err
}

func (c *HTTPClient) sendRequest(
	step *api.Step, timeout time.Duration, httpReq *http.Request,
) ([]byte, error) {
	ctx, cancel := context.WithTimeout(httpReq.Context(), timeout)
	defer cancel()

	req := httpReq.Clone(ctx)

	start := scheduler.Now()
	resp, err := c.httpClient.Do(req)
	dur := time.Since(start)

	if err != nil {
		slog.Error("HTTP request failed",
			log.StepID(step.ID),
			slog.Int("duration_ms", int(dur.Milliseconds())),
			log.Error(err))
		return nil, errors.Join(api.ErrWorkNotCompleted, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read response body",
			log.StepID(step.ID),
			log.Error(err))
		return nil, err
	}

	if resp.StatusCode/100 != 2 { // not a 2xx success response
		return nil, httpError(
			resp.StatusCode, resp.Header.Get("Content-Type"), respBody,
		)
	}

	return respBody, nil
}

func (c *HTTPClient) requestTimeout(timeoutMS int64) time.Duration {
	if timeoutMS > 0 {
		return time.Duration(timeoutMS) * time.Millisecond
	}
	return c.timeout
}

func buildRequest(
	step *api.Step, args api.Args, meta api.Metadata,
) (*http.Request, error) {
	method := step.HTTP.Invoke.DefaultedMethod()
	endpoint, err := resolveEndpoint(step.HTTP.Invoke.Endpoint, args)
	if err != nil {
		slog.Error("Failed to resolve HTTP endpoint",
			log.StepID(step.ID),
			log.Error(err))
		return nil, err
	}

	bodyReader, err := requestBody(method, args)
	if err != nil {
		slog.Error("Failed to marshal step request",
			log.StepID(step.ID),
			log.Error(err))
		return nil, err
	}

	httpReq, err := http.NewRequest(method, endpoint, bodyReader)
	if err != nil {
		slog.Error("Failed to create HTTP request",
			log.StepID(step.ID),
			log.Error(err))
		return nil, err
	}

	httpReq.Header.Set("Accept", api.JSONContentType)
	httpReq.Header.Set("User-Agent", UserAgent)
	api.SetMetadataHeaders(httpReq.Header, meta)
	if bodyReader != nil {
		httpReq.Header.Set("Content-Type", api.JSONContentType)
	}

	return httpReq, nil
}

func resolveEndpoint(endpoint string, args api.Args) (string, error) {
	matches := endpointParamPattern.FindAllStringSubmatchIndex(endpoint, -1)
	if len(matches) == 0 {
		return endpoint, nil
	}

	var buf bytes.Buffer
	last := 0
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		start := match[0]
		end := match[1]
		nameStart := match[2]
		nameEnd := match[3]
		name := api.Name(endpoint[nameStart:nameEnd])
		value, ok := args[name]
		if !ok {
			return "", fmt.Errorf("%w: %s", ErrMissingEndpointArg, name)
		}
		buf.WriteString(endpoint[last:start])
		buf.WriteString(url.PathEscape(endpointValue(value)))
		last = end
	}
	buf.WriteString(endpoint[last:])
	return buf.String(), nil
}

func endpointValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}

	data, err := json.Marshal(value)
	if err == nil {
		if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
			var s string
			if unmarshalErr := json.Unmarshal(data, &s); unmarshalErr == nil {
				return s
			}
		}
		return string(data)
	}

	return fmt.Sprint(value)
}

func requestBody(method string, args api.Args) (io.Reader, error) {
	if method == "GET" {
		return nil, nil
	}

	body, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(body), nil
}

func compensateRequestBody(
	method string, req CompensateRequest,
) (io.Reader, error) {
	if method == "GET" {
		return nil, nil
	}

	body, err := json.Marshal(
		compensateBody{Input: req.Inputs, Output: req.Outputs},
	)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(body), nil
}

func parseResponse(step *api.Step, respBody []byte) (api.Args, error) {
	if len(bytes.TrimSpace(respBody)) == 0 {
		return nil, nil
	}

	var outputs api.Args
	if err := json.Unmarshal(respBody, &outputs); err != nil {
		slog.Error("Failed to unmarshal response",
			log.StepID(step.ID),
			log.Error(err))
		return nil, fmt.Errorf("%w: %w", ErrInvalidOutputJSON, err)
	}

	return outputs, nil
}

func httpError(status int, contentType string, body []byte) error {
	problem := problemFromBody(contentType, body)
	err := httpStatusError(status, problem)
	if retryableHTTPStatus(status) {
		return errors.Join(api.ErrWorkNotCompleted, err)
	}
	return err
}

func httpStatusError(status int, problem *api.ProblemDetails) error {
	if problem != nil && problem.Error() != "" {
		return fmt.Errorf("%w: status %d: %s", ErrHTTPError, status, problem)
	}
	return fmt.Errorf("%w: status %d", ErrHTTPError, status)
}

func retryableHTTPStatus(status int) bool {
	return status >= http.StatusInternalServerError
}

func problemFromBody(contentType string, body []byte) *api.ProblemDetails {
	if !api.IsProblemJSON(contentType) {
		return nil
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	var problem api.ProblemDetails
	if err := json.Unmarshal(body, &problem); err != nil {
		return nil
	}
	if problem.Type == "" && problem.Title == "" && problem.Detail == "" {
		return nil
	}
	return &problem
}
