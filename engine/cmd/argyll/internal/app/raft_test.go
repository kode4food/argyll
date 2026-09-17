package app_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/kode4food/timebox/raft"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/cmd/argyll/internal/app"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/config"
)

func TestLoadRaftConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr error
		check   func(*testing.T, app.Config)
	}{
		{
			name: "defaults",
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, config.DefaultNodeID, c.Raft.LocalID)
				assert.Equal(t, app.DefaultRaftAddress, c.Raft.Address)
				assert.Equal(t, raft.DefaultLogTailSize, c.Raft.LogTailSize)
				assert.Len(t, c.Raft.Servers, 1)
				assert.Equal(t, config.DefaultNodeID, c.Raft.Servers[0].ID)
				assert.Equal(t,
					[]api.NodeID{config.DefaultNodeID}, c.Engine.Nodes,
				)
			},
		},
		{
			name: "load_raft_settings",
			envVars: map[string]string{
				"RAFT_NODE_ID":       "node-2",
				"RAFT_ADDRESS":       "10.0.0.2:9702",
				"RAFT_DATA_DIR":      "/tmp/argyll-node-2",
				"RAFT_LOG_TAIL_SIZE": "4096",
				"RAFT_SERVERS": "node-1=10.0.0.1:9701," +
					"node-2=10.0.0.2:9702",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, "node-2", c.Raft.LocalID)
				assert.Equal(t, "10.0.0.2:9702", c.Raft.Address)
				assert.Equal(t, "/tmp/argyll-node-2", c.Raft.DataDir)
				assert.Equal(t, 4096, c.Raft.LogTailSize)
				assert.Len(t, c.Raft.Servers, 2)
				assert.Equal(t, "node-1", c.Raft.Servers[0].ID)
				assert.Equal(t, "10.0.0.2:9702", c.Raft.Servers[1].Address)
				assert.Equal(t, api.NodeID("node-2"), c.Engine.NodeID)
				assert.Equal(t,
					[]api.NodeID{"node-1", "node-2"}, c.Engine.Nodes,
				)
			},
		},
		{
			name: "node_id_sets_default_data_dir",
			envVars: map[string]string{
				"RAFT_NODE_ID": "node-2",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, "node-2", c.Raft.LocalID)
				assert.Equal(t, filepath.Join(
					os.TempDir(), app.DefaultRaftDataDirName, "node-2",
				), c.Raft.DataDir)
				assert.Equal(t, "node-2", c.Raft.Servers[0].ID)
			},
		},
		{
			name: "invalid_raft_servers_errors",
			envVars: map[string]string{
				"RAFT_SERVERS": "node-1-missing-equals",
			},
			wantErr: app.ErrInvalidRaftServers,
		},
		{
			name: "invalid_raft_log_tail_size_errors",
			envVars: map[string]string{
				"RAFT_LOG_TAIL_SIZE": "0",
			},
			wantErr: app.ErrEnvOutOfRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.envVars)

			cfg, err := app.LoadConfig()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			tt.check(t, cfg)
		})
	}
}

func TestRaftStatus(t *testing.T) {
	addr := availableAddress(t)
	nid := "node-" + strconv.Itoa(availablePort(t))
	b, err := raft.Open(raft.DefaultConfig().With(raft.Config{
		LocalID: nid,
		Address: addr,
		DataDir: t.TempDir(),
		Servers: []raft.Server{{ID: nid, Address: addr}},
	}))
	assert.NoError(t, err)
	if b != nil {
		defer func() { _ = b.Close() }()
	}

	st := app.NewRaftStatusProvider(b)()
	if assert.Contains(t, st, "backend") {
		backend, ok := st["backend"].(map[string]any)
		if assert.True(t, ok) {
			assert.Equal(t, "raft", backend["type"])
			assert.NotEmpty(t, backend["state"])
			assert.Contains(t, backend, "leader_address")
			assert.Contains(t, backend, "leader_id")
		}
	}
}
