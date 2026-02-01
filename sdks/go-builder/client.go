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

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// Client provides functionality for interacting with the orchestrator
	// API, including step registration, flow management, and state queries
	Client struct {
		httpClient *http.Client
		baseURL    string
	}

	// FlowClient provides access to a specific flow
	FlowClient struct {
		*Client
		flowID api.FlowID
	}

	httpRequest struct {
		Method    string
		URL       string
		Body      any
		ErrorType error
		Accepted  []int
		Result    any
	}
)

var (
	ErrRegisterStep = errors.New("failed to register step")
	ErrUpdateStep   = errors.New("failed to update step")
	ErrListSteps    = errors.New("failed to list steps")
	ErrStartFlow    = errors.New("failed to start flow")
	ErrGetFlow      = errors.New("failed to get flow")
)

const (
	DefaultStepPort = 8081

	routeSteps = "/engine/step"
	routeFlow  = "/engine/flow"
)

// NewClient creates a new orchestrator client with the specified base URL and
// timeout
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ListSteps retrieves all registered steps from the orchestrator
func (c *Client) ListSteps(
	ctx context.Context,
) (*api.StepsListResponse, error) {
	var result api.StepsListResponse
	err := c.doHTTPRequest(ctx, httpRequest{
		Method:    "GET",
		URL:       c.url(routeSteps),
		ErrorType: ErrListSteps,
		Accepted:  []int{http.StatusOK},
		Result:    &result,
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Flow returns a client for accessing a specific flow
func (c *Client) Flow(id api.FlowID) *FlowClient {
	return &FlowClient{
		Client: c,
		flowID: id,
	}
}

func (c *Client) url(format string, args ...any) string {
	path := fmt.Sprintf(format, args...)
	return c.baseURL + path
}

func (c *Client) registerStep(ctx context.Context, step *api.Step) error {
	return c.doHTTPRequest(ctx, httpRequest{
		Method:    "POST",
		URL:       c.url(routeSteps),
		Body:      step,
		ErrorType: ErrRegisterStep,
		Accepted:  []int{http.StatusOK, http.StatusCreated},
	})
}

func (c *Client) updateStep(ctx context.Context, step *api.Step) error {
	return c.doHTTPRequest(ctx, httpRequest{
		Method:    "PUT",
		URL:       c.url("%s/%s", routeSteps, step.ID),
		Body:      step,
		ErrorType: ErrUpdateStep,
		Accepted:  []int{http.StatusOK},
	})
}

func (c *Client) startFlow(
	ctx context.Context, req *api.CreateFlowRequest,
) error {
	return c.doHTTPRequest(ctx, httpRequest{
		Method:    "POST",
		URL:       c.url(routeFlow),
		Body:      req,
		ErrorType: ErrStartFlow,
		Accepted:  []int{http.StatusOK, http.StatusCreated},
	})
}

// GetState retrieves the current state of the flow
func (c *FlowClient) GetState(ctx context.Context) (*api.FlowState, error) {
	var result api.FlowState
	err := c.doHTTPRequest(ctx, httpRequest{
		Method:    "GET",
		URL:       c.url("%s/%s", routeFlow, c.flowID),
		ErrorType: ErrGetFlow,
		Accepted:  []int{http.StatusOK},
		Result:    &result,
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// FlowID returns the flow ID for this client
func (c *FlowClient) FlowID() api.FlowID {
	return c.flowID
}

func (c *Client) doHTTPRequest(ctx context.Context, req httpRequest) error {
	var body io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return err
		}
		body = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, body)
	if err != nil {
		return err
	}

	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	for _, status := range req.Accepted {
		if resp.StatusCode == status {
			if req.Result != nil {
				return json.NewDecoder(resp.Body).Decode(req.Result)
			}
			return nil
		}
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("%w: status %d, body: %s",
		req.ErrorType, resp.StatusCode, string(respBody))
}
