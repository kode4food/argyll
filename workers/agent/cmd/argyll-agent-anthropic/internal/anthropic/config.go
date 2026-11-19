package anthropic

import (
	_ "embed"
	"encoding/json"
	"os"

	"github.com/kode4food/argyll/engine/pkg/api"
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

func loadPrefixThresholds() (map[api.ModelID]int, error) {
	var thresholds map[api.ModelID]int
	if err := json.Unmarshal([]byte(defaultPrefixThresholds), &thresholds); err != nil {
		return nil, err
	}
	if err := worker.ReadJSON(os.Getenv("PREFIX_THRESHOLDS"), &thresholds); err != nil {
		return nil, err
	}
	return thresholds, nil
}
