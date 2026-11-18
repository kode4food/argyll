package openai

import (
	_ "embed"

	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

//go:embed model-prices.json
var defaultPrices string

// LoadConfig loads the worker configuration with OpenAI model prices as defaults
func LoadConfig() (*worker.Config, error) {
	return worker.LoadConfig(defaultPrices)
}
