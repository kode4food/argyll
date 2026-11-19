package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"regexp"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	// Config is what a worker offers: the MCP servers an Agent may use, the base URLs of the
	// providers it reaches, and the model prices behind the cost it reports
	Config struct {
		Servers   map[string]MCPServer
		Providers map[api.ProviderID]string
		Prices    map[ModelKey]Price
	}

	// MCPServer is an MCP server reached over streamable HTTP
	MCPServer struct {
		Headers map[string]string `json:"headers"`
		URL     string            `json:"url"`
	}

	// ModelKey identifies a model by its provider and model ID
	ModelKey struct {
		Provider api.ProviderID
		Model    api.ModelID
	}

	// Price defines the base rates and any input-length tiers for a model
	Price struct {
		Rates
		Tiers []PriceTier `json:"tiers,omitempty"`
	}

	// PriceTier applies to a turn with more than InputOver total input tokens
	PriceTier struct {
		Rates
		InputOver int64 `json:"input_over"`
	}

	// Rates holds per-million-token charges for one pricing tier
	Rates struct {
		CacheInput  *float64 `json:"cache_input,omitempty"`
		CacheOutput *float64 `json:"cache_output,omitempty"`
		Input       *float64 `json:"input,omitempty"`
		Output      *float64 `json:"output,omitempty"`
	}
)

var (
	ErrInvalidServerName = errors.New("invalid MCP server name")
	ErrMissingServerURL  = errors.New("MCP server needs a url")
	ErrMissingEnv        = errors.New("environment variable not set")
	ErrInvalidProviders  = errors.New("invalid PROVIDER_URLS")
	ErrInvalidModelKey   = errors.New("invalid provider/model key")
	ErrInvalidPriceTier  = errors.New("invalid model price tier")
)

// LoadConfig combines embedded model prices with worker settings from the environment. Headers
// in the MCP server file may reference environment variables as ${VAR}
func LoadConfig(defaultPrices string) (*Config, error) {
	servers, err := loadServers(os.Getenv("MCP_SERVERS"))
	if err != nil {
		return nil, err
	}
	url := os.Getenv("ARGYLL_MCP_URL")
	if _, ok := servers["argyll"]; !ok && url != "" {
		servers["argyll"] = MCPServer{URL: url}
	}
	prices, err := loadPrices(defaultPrices, os.Getenv("MODEL_PRICES"))
	if err != nil {
		return nil, err
	}
	providers, err := parseProviders(os.Getenv("PROVIDER_URLS"))
	if err != nil {
		return nil, err
	}
	return &Config{
		Servers:   servers,
		Providers: providers,
		Prices:    prices,
	}, nil
}

// ReadJSON decodes an optional JSON configuration file
func ReadJSON(path string, v any) error {
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// UnmarshalText reads a model key from a JSON price table
func (k *ModelKey) UnmarshalText(text []byte) error {
	provider, model, ok := strings.Cut(string(text), "/")
	if !ok || provider == "" || model == "" {
		return fmt.Errorf("%w: %q", ErrInvalidModelKey, text)
	}
	*k = ModelKey{
		Provider: api.ProviderID(provider),
		Model:    api.ModelID(model),
	}
	return nil
}

func (p Price) ratesFor(input int64) Rates {
	rates := p.Rates
	var threshold int64
	for _, tier := range p.Tiers {
		if input > tier.InputOver && tier.InputOver > threshold {
			rates = tier.Rates
			threshold = tier.InputOver
		}
	}
	return rates
}

func loadPrices(defaults, path string) (map[ModelKey]Price, error) {
	prices := map[ModelKey]Price{}
	if err := json.Unmarshal([]byte(defaults), &prices); err != nil {
		return nil, err
	}
	var overrides map[ModelKey]Price
	if err := ReadJSON(path, &overrides); err != nil {
		return nil, err
	}
	if prices == nil {
		prices = map[ModelKey]Price{}
	}
	maps.Copy(prices, overrides)
	for key, price := range prices {
		seen := util.Set[int64]{}
		for _, tier := range price.Tiers {
			if tier.InputOver <= 0 || seen.Contains(tier.InputOver) {
				return nil, fmt.Errorf(
					"%w: %s/%s: %d", ErrInvalidPriceTier, key.Provider, key.Model, tier.InputOver,
				)
			}
			seen.Add(tier.InputOver)
		}
	}
	return prices, nil
}

func loadServers(path string) (map[string]MCPServer, error) {
	raw := map[string]MCPServer{}
	if err := ReadJSON(path, &raw); err != nil {
		return nil, err
	}
	res := make(map[string]MCPServer, len(raw))
	serverName := regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
	for name, s := range raw {
		if !serverName.MatchString(name) {
			return nil, fmt.Errorf("%w: %q", ErrInvalidServerName, name)
		}
		if s.URL == "" {
			return nil, fmt.Errorf("%w: %s", ErrMissingServerURL, name)
		}
		headers, err := expandHeaders(s.Headers)
		if err != nil {
			return nil, err
		}
		res[name] = MCPServer{
			Headers: headers,
			URL:     s.URL,
		}
	}
	return res, nil
}

func expandHeaders(raw map[string]string) (map[string]string, error) {
	headers := make(map[string]string, len(raw))
	for k, v := range raw {
		h, err := expandEnv(v)
		if err != nil {
			return nil, err
		}
		headers[k] = h
	}
	return headers, nil
}

// parseProviders reads comma-separated provider=url pairs
func parseProviders(value string) (map[api.ProviderID]string, error) {
	res := map[api.ProviderID]string{}
	for part := range strings.SplitSeq(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		provider, url, ok := strings.Cut(part, "=")
		provider = strings.TrimSpace(provider)
		url = strings.TrimSpace(url)
		if !ok || provider == "" || url == "" {
			return nil, fmt.Errorf("%w: %q", ErrInvalidProviders, part)
		}
		res[api.ProviderID(provider)] = url
	}
	return res, nil
}

func expandEnv(s string) (string, error) {
	var missing string
	res := os.Expand(s, func(key string) string {
		v, ok := os.LookupEnv(key)
		if !ok && missing == "" {
			missing = key
		}
		return v
	})
	if missing != "" {
		return "", fmt.Errorf("%w: %s", ErrMissingEnv, missing)
	}
	return res, nil
}
