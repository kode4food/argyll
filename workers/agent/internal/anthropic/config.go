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
	cfg, err := worker.LoadConfig(worker.Defaults{
		Prices: defaultPrices,
	})
	if err != nil {
		return nil, nil, err
	}
	p, err := NewProvider(cfg.Providers)
	return cfg, p, err
}

func loadPrefixThresholds() (map[worker.ModelKey]int, error) {
	var thresholds map[worker.ModelKey]int
	if err := json.Unmarshal([]byte(defaultPrefixThresholds), &thresholds); err != nil {
		return nil, err
	}
	if err := worker.ReadJSON(os.Getenv("PREFIX_THRESHOLDS"), &thresholds); err != nil {
		return nil, err
	}
	return thresholds, nil
}
