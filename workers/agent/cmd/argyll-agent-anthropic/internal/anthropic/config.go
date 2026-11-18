package anthropic

import (
	_ "embed"
	"encoding/json"
	"os"

	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

var (
	//go:embed model-prices.json
	defaultPrices string

	//go:embed prefix-thresholds.json
	defaultPrefixThresholds string
)

// LoadConfig loads the Anthropic worker and provider configuration
func LoadConfig() (*worker.Config, *Provider, error) {
	cfg, err := worker.LoadConfig(defaultPrices)
	if err != nil {
		return nil, nil, err
	}
	p, err := New(cfg.Providers)
	return cfg, p, err
}

func loadPrefixThresholds() (map[string]int, error) {
	var thresholds map[string]int
	if err := json.Unmarshal([]byte(defaultPrefixThresholds), &thresholds); err != nil {
		return nil, err
	}
	path := os.Getenv("PREFIX_THRESHOLDS")
	if path == "" {
		return thresholds, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &thresholds); err != nil {
		return nil, err
	}
	return thresholds, nil
}
