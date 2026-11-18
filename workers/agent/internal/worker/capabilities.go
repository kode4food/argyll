package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"sync"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func (w *Worker) capabilities(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), DefaultListTimeout)
	defer cancel()

	names := w.provider.Names()
	providers := make([]api.AgentProvider, len(names))
	var servers []api.MCPServer
	var wg sync.WaitGroup
	if len(names) > 0 {
		wg.Go(func() { servers = listServers(ctx, w.cfg.Servers) })
	}
	for i, name := range names {
		wg.Go(func() {
			models, err := w.provider.Models(ctx, name)
			if err != nil {
				slog.Warn("Agent model discovery failed", slog.String("provider", name))
			}
			slices.Sort(models)
			providers[i] = api.AgentProvider{
				Provider: name,
				Models:   slices.Compact(models),
			}
		})
	}
	wg.Wait()
	for i := range providers {
		providers[i].MCP = servers
	}
	rw.Header().Set("Content-Type", api.JSONContentType)
	_ = json.NewEncoder(rw).Encode(api.AgentCapabilities{Providers: providers})
}
