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

	respBody, err := c.sendAction(sendActionArgs{
		step:    step,
		action:  &step.HTTP.Invoke,
		name:    "invoke",
		args:    args,
		meta:    meta,
		timeout: step.HTTP.Invoke.Timeout,
	})
	if err != nil {
		return nil, err
	}

	return parseResponse(step, respBody)
}

// InvokeCompensate sends selected work attributes to the compensate endpoint
func (c *HTTPClient) InvokeCompensate(req CompensateRequest) error {
	step := req.Step
	if step.HTTP == nil || step.HTTP.Compensate == nil {
		return fmt.Errorf("%w: %s", ErrNoHTTPConfig, step.ID)
	}

	args, err := buildCompensationArgs(req)
	if err != nil {
		return err
	}

	_, err = c.sendAction(sendActionArgs{
		step:    step,
		action:  step.HTTP.Compensate,
		name:    "compensate",
		args:    args,
		meta:    req.Metadata,
		timeout: step.HTTP.CompensateTimeout(),
	})
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

// sendActionArgs describes one HTTP action call
type sendActionArgs struct {
	step    *api.Step
	action  *api.HTTPAction
	name    string
	args    api.Args
	meta    api.Metadata
	timeout int64
}

func (c *HTTPClient) sendAction(a sendActionArgs) ([]byte, error) {
	endpoint, err := resolveEndpoint(a.action.Endpoint, a.args)
	if err != nil {
		slog.Error("Failed to resolve endpoint",
			log.StepID(a.step.ID),
			slog.String("action", a.name),
			log.Error(err))
		return nil, err
	}

	method := a.action.DefaultedMethod()
	var body io.Reader
	if method != "GET" && method != "DELETE" {
		data, err := json.Marshal(a.args)
		if err != nil {
			slog.Error("Failed to marshal request",
				log.StepID(a.step.ID),
				slog.String("action", a.name),
				log.Error(err))
			return nil, err
		}
		body = bytes.NewBuffer(data)
	}

	httpReq, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		slog.Error("Failed to create request",
			log.StepID(a.step.ID),
			slog.String("action", a.name),
			log.Error(err))
		return nil, err
	}

	httpReq.Header.Set("Accept", api.JSONContentType)
	httpReq.Header.Set("User-Agent", UserAgent)
	api.SetMetadataHeaders(httpReq.Header, a.meta)
	if body != nil {
		httpReq.Header.Set("Content-Type", api.JSONContentType)
	}

	return c.sendRequest(a.step, c.requestTimeout(a.timeout), httpReq)
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

func buildCompensationArgs(req CompensateRequest) (api.Args, error) {
	res := api.Args{}
	for name, attr := range req.Step.Attributes {
		if attr == nil || !attr.Compensated {
			continue
		}
		mapped, _ := req.Step.MappedName(name)
		source := req.Inputs
		if attr.IsOutput() {
			source = req.Outputs
		}
		value, ok := source[mapped]
		if !ok {
			continue
		}
		if _, ok := res[mapped]; ok {
			return nil, fmt.Errorf(
				"%w: %s", api.ErrCompensateArgConflict, mapped,
			)
		}
		res[mapped] = value
	}
	return res, nil
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
