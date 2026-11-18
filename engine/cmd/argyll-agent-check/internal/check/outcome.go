package check

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// callback serves the webhook URL a live attempt reports its outcome to
	callback struct {
		server   *http.Server
		url      string
		outcomes chan outcome
	}

	outcome struct {
		value api.AgentOutcome
		err   error
	}
)

const (
	callbackPath = "/callback"
	inputTokens  = "input_tokens"
	outputTokens = "output_tokens"
)

func (c *callback) receive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != callbackPath {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	out := readOutcome(r.Body)
	select {
	case c.outcomes <- out:
	default:
	}
	if out.err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *callback) wait(timeout time.Duration) (api.AgentOutcome, error) {
	select {
	case out := <-c.outcomes:
		return out.value, out.err
	case <-time.After(timeout):
		return api.AgentOutcome{}, fmt.Errorf("%w: %s", ErrNoOutcome, timeout)
	}
}

func (c *callback) close() {
	_ = c.server.Close()
}

func readOutcome(r io.Reader) outcome {
	var v api.AgentOutcome
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return outcome{err: fmt.Errorf("%w: %w", ErrInvalidOutcome, err)}
	}
	if (v.Outputs == nil) == (v.Problem == nil) {
		return outcome{err: ErrInvalidOutcome}
	}
	return outcome{value: v}
}

func listen(cfg *Config) (*callback, error) {
	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return nil, err
	}
	base := cfg.CallbackURL
	if base == "" {
		base = "http://" + ln.Addr().String()
	}
	cb := &callback{
		url:      base + callbackPath,
		outcomes: make(chan outcome, 1),
	}
	cb.server = &http.Server{
		Handler:           http.HandlerFunc(cb.receive),
		ReadHeaderTimeout: DefaultRequestTimeout,
	}
	go func() { _ = cb.server.Serve(ln) }()
	return cb, nil
}

func outcomeErr(out api.AgentOutcome, err error) error {
	if err != nil {
		return err
	}
	if p := out.Problem; p != nil {
		return fmt.Errorf("%w: %d: %w", ErrProblemOutcome, p.Status, p)
	}
	if _, ok := out.Outputs[outputName]; !ok || len(out.Outputs) != 1 {
		return ErrOutputMismatch
	}
	return nil
}

func usageErr(out api.AgentOutcome, err error) error {
	if err != nil {
		return err
	}
	usage := out.Usage
	_, in := usage[inputTokens]
	_, gen := usage[outputTokens]
	if !in || !gen {
		return fmt.Errorf("%w: %v", ErrMissingUsage, usage)
	}
	return nil
}
