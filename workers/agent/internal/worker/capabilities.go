package worker

import (
	"cmp"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type catalogCache struct {
	mu      sync.Mutex
	value   api.AgentCapabilities
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

func (w *Worker) loadCapabilities(ctx context.Context) api.AgentCapabilities {
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

func (w *Worker) discoverCapabilities(ctx context.Context) (api.AgentCapabilities, bool) {
	names := w.provider.Names()
	providers := make([]api.AgentProvider, len(names))
	var servers []api.MCPServer
	var wg sync.WaitGroup
	var modelFailure atomic.Bool
	if len(names) > 0 {
		wg.Go(func() { servers = listServers(ctx, w.cfg.Servers) })
	}
	for i, name := range names {
		wg.Go(func() {
			models, err := w.provider.Models(ctx, name)
			if err != nil {
				slog.Warn("Agent model discovery failed", slog.String("provider", string(name)))
				modelFailure.Store(true)
			}
			if models == nil {
				models = []api.AgentModel{}
			}
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
	serverFailure := slices.ContainsFunc(servers, func(s api.MCPServer) bool {
		return s.Tools == nil
	})
	return api.AgentCapabilities{Providers: providers}, !modelFailure.Load() && !serverFailure
}
