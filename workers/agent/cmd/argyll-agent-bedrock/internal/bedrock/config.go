package bedrock

import (
	_ "embed"

	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

var (
	//go:embed model-prices.json
	defaultPrices string

	//go:embed model-capabilities.json
	defaultModelCapabilities string
)

// LoadConfig loads the Bedrock worker and provider configuration
func LoadConfig() (*worker.Config, *Provider, error) {
	cfg, err := worker.LoadConfig(worker.Defaults{
		Prices:       defaultPrices,
		Capabilities: defaultModelCapabilities,
	})
	if err != nil {
		return nil, nil, err
	}
	p, err := NewProvider(cfg)
	return cfg, p, err
}
