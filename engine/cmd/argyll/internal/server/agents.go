package server

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
	"github.com/kode4food/argyll/engine/pkg/util"
)

// DefaultAgentListTimeout bounds discovery of Agent capabilities
const DefaultAgentListTimeout = 15 * time.Second

var (
	ErrAgentCapabilities = errors.New("agent worker capabilities unavailable")
)

// RegisterAgentRoutes serves the Agent capabilities available through the configured workers
func RegisterAgentRoutes(router *gin.Engine, workers builtins.AgentWorkers) {
	client := &http.Client{Timeout: DefaultAgentListTimeout}
	router.GET("/engine/agents/capabilities", func(c *gin.Context) {
		res, err := listAgentCapabilities(c.Request.Context(), client, workers)
		if err != nil {
			c.JSON(http.StatusBadGateway, api.ErrorResponse{
				Error:  err.Error(),
				Status: http.StatusBadGateway,
			})
			return
		}
		c.JSON(http.StatusOK, res)
	})
}

func listAgentCapabilities(
	ctx context.Context, client *http.Client, workers builtins.AgentWorkers,
) (*api.AgentCapabilities, error) {
	urls := util.Set[string]{}
	for _, base := range workers {
		urls.Add(base)
	}
	res := &api.AgentCapabilities{Providers: []api.AgentProvider{}}
	seen := util.Set[api.ProviderID]{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var answered bool
	for base := range urls {
		wg.Go(func() {
			providers, err := fetchAgentProviders(ctx, client, base)
			if err != nil {
				slog.Warn("Agent worker discovery failed", log.Error(err))
				return
			}
			mu.Lock()
			defer mu.Unlock()
			answered = true
			for _, provider := range providers {
				url, ok := workers.URL(provider.Provider)
				if !ok || url != base || seen.Contains(provider.Provider) ||
					provider.Provider == "" || provider.Provider == builtins.AnyProvider {
					continue
				}
				res.Providers = append(res.Providers, provider)
				seen.Add(provider.Provider)
			}
		})
	}
	wg.Wait()
	if !urls.IsEmpty() && !answered {
		return nil, ErrAgentCapabilities
	}
	slices.SortFunc(res.Providers, func(a, b api.AgentProvider) int {
		return cmp.Compare(a.Provider, b.Provider)
	})
	return res, nil
}

func fetchAgentProviders(
	ctx context.Context, client *http.Client, base string,
) ([]api.AgentProvider, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, base+builtins.AgentCapabilitiesPath, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAgentCapabilities, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAgentCapabilities, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrAgentCapabilities, resp.StatusCode)
	}
	var res api.AgentCapabilities
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAgentCapabilities, err)
	}
	return res.Providers, nil
}
