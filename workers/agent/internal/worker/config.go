package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type (
	// Config is what a worker offers: the MCP servers an Agent may use, the base URLs of the
	// providers it reaches, and the model prices behind the cost it reports
	Config struct {
		Servers   map[string]MCPServer
		Providers map[string]string
		Prices    map[string]Price
	}

	// MCPServer is an MCP server reached over streamable HTTP
	MCPServer struct {
		Headers map[string]string `json:"headers"`
		URL     string            `json:"url"`
	}

	// Price is what a model charges per million tokens
	Price struct {
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
	var prices map[string]Price
	if err := json.Unmarshal([]byte(defaultPrices), &prices); err != nil {
		return nil, err
	}
	if err := readJSON(os.Getenv("MODEL_PRICES"), &prices); err != nil {
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

func loadServers(path string) (map[string]MCPServer, error) {
	raw := map[string]MCPServer{}
	if err := readJSON(path, &raw); err != nil {
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
func parseProviders(value string) (map[string]string, error) {
	res := map[string]string{}
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
		res[provider] = url
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

func readJSON(path string, v any) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
