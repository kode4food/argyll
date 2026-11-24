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
	"github.com/kode4food/argyll/engine/pkg/auth"
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
func RegisterAgentRoutes(
	router *gin.Engine, workers builtins.AgentWorkers, a auth.Authenticator,
) {
	client := &http.Client{Timeout: DefaultAgentListTimeout}
	router.GET("/engine/agents/capabilities",
		requireRole(auth.RoleReader, a),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, listAgentCapabilities(c.Request.Context(), client, workers))
		},
	)
}

func listAgentCapabilities(
	ctx context.Context, client *http.Client, workers builtins.AgentWorkers,
) *api.AgentCapabilities {
	urls := util.Set[string]{}
	for _, base := range workers {
		urls.Add(base)
	}
	res := &api.AgentCapabilities{Providers: []api.AgentProvider{}}
	seen := util.Set[api.ProviderID]{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for base := range urls {
		wg.Go(func() {
			caps, err := fetchAgentCapabilities(ctx, client, base)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				slog.Warn("Agent worker discovery failed",
					slog.String("worker", base), log.Error(err))
				p := api.NewProblem(http.StatusBadGateway, err.Error())
				if cause, ok := errors.AsType[*api.ProblemDetails](err); ok {
					*p = *cause
				}
				res.Problems = append(res.Problems, *p)
				return
			}
			res.Problems = append(res.Problems, caps.Problems...)
			for _, provider := range caps.Providers {
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
	slices.SortFunc(res.Providers, func(a, b api.AgentProvider) int {
		return cmp.Compare(a.Provider, b.Provider)
	})
	return res
}

func fetchAgentCapabilities(
	ctx context.Context, client *http.Client, base string,
) (*api.AgentCapabilities, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, base+builtins.AgentCapabilitiesPath, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrAgentCapabilities, base, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrAgentCapabilities, base, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Join(ErrAgentCapabilities, api.ReadHTTPProblem(resp))
	}
	var res api.AgentCapabilities
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrAgentCapabilities, base, err)
	}
	return &res, nil
}
