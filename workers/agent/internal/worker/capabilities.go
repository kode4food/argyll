package worker

import (
	"cmp"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type catalogCache struct {
	mu      sync.Mutex
	value   *api.AgentCapabilities
	expires time.Time
}

const (
	capabilitiesTTL        = time.Hour
	partialCapabilitiesTTL = 15 * time.Second
)

func (w *Worker) capabilities(rw http.ResponseWriter, r *http.Request) {
	res := w.loadCapabilities(r.Context())
	rw.Header().Set("Content-Type", api.JSONContentType)
	_ = json.NewEncoder(rw).Encode(res)
}

func (w *Worker) loadCapabilities(ctx context.Context) *api.AgentCapabilities {
	cache := &w.catalog
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if time.Now().Before(cache.expires) {
		return cache.value
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), DefaultListTimeout)
	defer cancel()
	res, complete := w.discoverCapabilities(ctx)
	ttl := capabilitiesTTL
	if !complete {
		ttl = partialCapabilitiesTTL
	}
	cache.value = res
	cache.expires = time.Now().Add(ttl)
	return res
}

func (w *Worker) discoverCapabilities(ctx context.Context) (*api.AgentCapabilities, bool) {
	names := w.provider.Names()
	providers := make([]api.AgentProvider, len(names))
	problems := make([]*api.ProblemDetails, len(names))
	var servers []api.MCPServer
	var failures []api.ProblemDetails
	var wg sync.WaitGroup
	if len(names) > 0 {
		wg.Go(func() { servers, failures = listServers(ctx, w.cfg.Servers) })
	}
	for i, name := range names {
		wg.Go(func() {
			models, err := w.provider.Models(ctx, name)
			if err != nil {
				slog.Warn("Agent model discovery failed",
					slog.String("provider", string(name)), log.Error(err))
				p := *problemOf(err)
				p.Instance = "provider/" + string(name)
				problems[i] = &p
			}
			models = w.listedModels(name, models)
			slices.SortFunc(models, func(a, b api.AgentModel) int {
				return cmp.Compare(a.ID, b.ID)
			})
			providers[i] = api.AgentProvider{
				Provider: name,
				Models: slices.CompactFunc(models, func(a, b api.AgentModel) bool {
					return a.ID == b.ID
				}),
			}
		})
	}
	wg.Wait()
	for i := range providers {
		providers[i].MCP = servers
	}
	for _, p := range problems {
		if p != nil {
			failures = append(failures, *p)
		}
	}
	return &api.AgentCapabilities{
		Providers: providers,
		Problems:  failures,
	}, len(failures) == 0
}

// listedModels applies the configured capabilities: hidden models are left out, and configured
// effort levels replace the provider's
func (w *Worker) listedModels(provider api.ProviderID, models []api.AgentModel) []api.AgentModel {
	res := make([]api.AgentModel, 0, len(models))
	for _, m := range models {
		caps := w.cfg.Capabilities[ModelKey{
			Provider: provider,
			Model:    m.ID,
		}]
		if caps.Hidden {
			continue
		}
		if len(caps.Efforts) > 0 {
			m.Efforts = caps.Efforts
		}
		res = append(res, m)
	}
	return res
}
