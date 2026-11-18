package app

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

var (
	ErrInvalidAgentWorkers = errors.New("invalid AGENT_WORKERS")
)

// loadAgentWorkers reads AGENT_WORKERS, a comma-separated list of provider=url pairs where the
// provider * names the worker serving every provider not listed
func loadAgentWorkers() (builtins.AgentWorkers, error) {
	res := builtins.AgentWorkers{}
	for part := range strings.SplitSeq(os.Getenv("AGENT_WORKERS"), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		provider, url, ok := strings.Cut(part, "=")
		provider = strings.TrimSpace(provider)
		url = strings.TrimRight(strings.TrimSpace(url), "/")
		if !ok || provider == "" || url == "" {
			return nil, fmt.Errorf("%w: %q", ErrInvalidAgentWorkers, part)
		}
		res[provider] = url
	}
	return res, nil
}
