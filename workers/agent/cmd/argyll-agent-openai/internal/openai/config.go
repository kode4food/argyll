package openai

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type (
	modelCapabilities struct {
		Efforts  []api.Effort  `json:"efforts"`
		Endpoint modelEndpoint `json:"endpoint"`
	}

	modelEndpoint string
)

const (
	chatCompletionsEndpoint modelEndpoint = "chat_completions"
	responsesEndpoint       modelEndpoint = "responses"
)

var (
	ErrInvalidModelEndpoint = errors.New("invalid model endpoint")
)

var (
	//go:embed model-prices.json
	defaultPrices string

	//go:embed model-capabilities.json
	defaultModelCapabilities string
)

// LoadConfig loads the worker and provider configuration
func LoadConfig() (*worker.Config, *Provider, error) {
	cfg, err := worker.LoadConfig(defaultPrices)
	if err != nil {
		return nil, nil, err
	}
	p, err := New(cfg.Providers)
	return cfg, p, err
}

func loadModelCapabilities(
	urls map[api.ProviderID]string,
) (map[worker.ModelKey]modelCapabilities, error) {
	defaults := defaultModelCapabilities
	if urls[Name] != "" {
		defaults = "{}"
	}
	var models map[worker.ModelKey]modelCapabilities
	if err := json.Unmarshal([]byte(defaults), &models); err != nil {
		return nil, err
	}
	var overrides map[worker.ModelKey]modelCapabilities
	if err := worker.ReadJSON(os.Getenv("MODEL_CAPABILITIES"), &overrides); err != nil {
		return nil, err
	}
	maps.Copy(models, overrides)
	for key, model := range models {
		switch model.Endpoint {
		case "", chatCompletionsEndpoint, responsesEndpoint:
		default:
			return nil, fmt.Errorf(
				"%w: %s/%s: %q", ErrInvalidModelEndpoint, key.Provider, key.Model, model.Endpoint,
			)
		}
	}
	return models, nil
}
