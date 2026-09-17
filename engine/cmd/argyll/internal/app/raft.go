package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kode4food/timebox/raft"

	"github.com/kode4food/argyll/engine/cmd/argyll/internal/server"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/config"
)

const (
	DefaultRaftAddress     = "127.0.0.1:9701"
	DefaultRaftDataDirName = "argyll-raft"
	MaxRaftLogTailSize     = 1_024_000
)

var (
	ErrInvalidRaftServers = errors.New("invalid RAFT_SERVERS")
)

// NewRaftStatusProvider reports the raft backend's role and leader to the
// server's health and status endpoints
func NewRaftStatusProvider(b *raft.Backend) server.StatusProvider {
	return func() map[string]any {
		addr, id := b.LeaderWithID()
		return map[string]any{
			"backend": map[string]any{
				"type":           "raft",
				"state":          b.State(),
				"leader_address": addr,
				"leader_id":      id,
			},
		}
	}
}

// loadRaftConfig reads the raft backend settings for this node from the
// environment. A node with no RAFT_SERVERS forms a cluster of one
func loadRaftConfig() (raft.Config, error) {
	nid := config.DefaultNodeID
	if nodeID := os.Getenv("RAFT_NODE_ID"); nodeID != "" {
		nid = nodeID
	}
	cfg := raft.DefaultConfig().With(raft.Config{
		LocalID: nid,
		Address: DefaultRaftAddress,
		DataDir: defaultRaftDataDir(nid),
	})
	if address := os.Getenv("RAFT_ADDRESS"); address != "" {
		cfg.Address = address
	}
	if dataDir := os.Getenv("RAFT_DATA_DIR"); dataDir != "" {
		cfg.DataDir = dataDir
	}
	if err := loadEnvInt(
		"RAFT_LOG_TAIL_SIZE", &cfg.LogTailSize, 0, MaxRaftLogTailSize,
	); err != nil {
		return raft.Config{}, err
	}

	cfg.Servers = []raft.Server{cfg.LocalServer()}
	if servers := os.Getenv("RAFT_SERVERS"); servers != "" {
		srvs, err := parseRaftServers(servers)
		if err != nil {
			return raft.Config{}, err
		}
		cfg.Servers = srvs
	}
	return cfg, cfg.Validate()
}

func raftNodes(srvs []raft.Server) []api.NodeID {
	res := make([]api.NodeID, 0, len(srvs))
	for _, srv := range srvs {
		res = append(res, api.NodeID(srv.ID))
	}
	return res
}

func defaultRaftDataDir(localID string) string {
	return filepath.Join(
		os.TempDir(), DefaultRaftDataDirName, localID,
	)
}

func parseRaftServers(spec string) ([]raft.Server, error) {
	parts := strings.Split(spec, ",")
	res := make([]raft.Server, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, addr, ok := strings.Cut(part, "=")
		id = strings.TrimSpace(id)
		addr = strings.TrimSpace(addr)
		if !ok || id == "" || addr == "" {
			return nil, fmt.Errorf("%w: %q", ErrInvalidRaftServers, part)
		}
		res = append(res, raft.Server{
			ID:      id,
			Address: addr,
		})
	}
	return res, nil
}

func formatRaftServers(srvs []raft.Server) string {
	parts := make([]string, 0, len(srvs))
	for _, srv := range srvs {
		parts = append(parts, srv.ID+"="+srv.Address)
	}
	return strings.Join(parts, ",")
}
