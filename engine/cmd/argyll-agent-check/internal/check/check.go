package check

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/auth"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

type (
	// Config names the worker under check, the model its live attempt runs, and where the worker
	// reports that attempt's outcome
	Config struct {
		WorkerURL   string
		Provider    string
		Model       string
		Listen      string
		CallbackURL string
		Timeout     time.Duration
	}

	// Result is the outcome of one check, which failed when Err is set
	Result struct {
		Name string
		Err  error
	}

	checker struct {
		cfg    *Config
		client *http.Client
	}
)

const (
	// DefaultRequestTimeout bounds each request the checker sends the worker
	DefaultRequestTimeout = 10 * time.Second

	// DefaultReportGrace is how long past the attempt's timeout the checker waits for its outcome
	DefaultReportGrace = 5 * time.Second

	// DefaultMaxTurns and DefaultMaxTokens leave a model room to call finish after a stray turn
	DefaultMaxTurns  = 3
	DefaultMaxTokens = 256

	checkID    = "argyll-agent-check"
	inputName  = "text"
	outputName = "echo"
	prologue   = "# Echo\n\nReturn the input `text` unchanged as the output `echo`."
)

var (
	ErrUnhealthy      = errors.New("worker is not healthy")
	ErrNotAccepted    = errors.New("worker did not accept the attempt")
	ErrNoOutcome      = errors.New("worker reported no outcome in time")
	ErrProblemOutcome = errors.New("attempt failed")
	ErrInvalidOutcome = errors.New("outcome is malformed")
	ErrOutputMismatch = errors.New("outputs differ from the requested names")
	ErrMissingUsage   = errors.New("usage lacks token counts")
)

// Run checks a worker's health and runs one attempt through the callback contract
func Run(cfg *Config) ([]Result, error) {
	cb, err := listen(cfg)
	if err != nil {
		return nil, err
	}
	defer cb.close()

	c := &checker{
		cfg: cfg,
		client: &http.Client{
			Timeout:       DefaultRequestTimeout,
			CheckRedirect: api.SameOriginRedirect,
		},
	}
	// A worker that checks tokens accepts the engine's service account, which the check borrows
	if ts := auth.ServiceTokens(); ts != nil {
		c.client.Transport = &oauth2.Transport{Source: ts}
	}
	res := []Result{result("health", c.health())}

	if err := c.accepts(c.request(), cb.url); err != nil {
		return append(res, result("accepts attempt", err)), nil
	}
	out, err := cb.wait(cfg.Timeout + DefaultReportGrace)
	return append(
		res, result("accepts attempt", nil),
		result("reports outputs", outcomeErr(out, err)),
		result("reports usage", usageErr(out, err)),
	), nil
}

func (c *checker) health() error {
	resp, err := c.client.Get(c.cfg.WorkerURL + builtins.AgentHealthPath)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: %s", ErrUnhealthy, resp.Status)
	}
	return nil
}

func (c *checker) accepts(body api.AgentRequest, webhook string) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := c.cfg.WorkerURL + builtins.AgentInvokePath
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", api.JSONContentType)
	req.Header.Set(api.HeaderFlowID, checkID)
	req.Header.Set(api.HeaderStepID, checkID)
	req.Header.Set(api.HeaderWorkToken, checkID)
	req.Header.Set(api.HeaderWebhookURL, webhook)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: %s", ErrNotAccepted, resp.Status)
	}
	return nil
}

func (c *checker) request() api.AgentRequest {
	return api.AgentRequest{
		Step: &api.Step{
			ID:   checkID,
			Name: checkID,
			Type: api.StepTypeAgent,
			Agent: &api.AgentConfig{
				Provider:  api.ProviderID(c.cfg.Provider),
				Model:     api.ModelID(c.cfg.Model),
				Prologue:  prologue,
				MaxTurns:  DefaultMaxTurns,
				MaxTokens: DefaultMaxTokens,
			},
			Attributes: api.AttributeSpecs{
				inputName:  {Role: api.RoleRequired},
				outputName: {Role: api.RoleOutput},
			},
		},
		Inputs:  api.Args{inputName: "argyll"},
		Timeout: c.cfg.Timeout.Milliseconds(),
	}
}

func result(name string, err error) Result {
	return Result{
		Name: name,
		Err:  err,
	}
}
