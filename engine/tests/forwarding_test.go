package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/raft"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type raftNode struct {
	id          string
	engStore    *timebox.Store
	flowStore   *timebox.Store
	persistence *raft.Persistence
	engine      *engine.Engine
	server      *server.Server
}

type raftInit struct {
	id  string
	cfg *config.Config
	hub *event.Hub
}

func TestFollowerWrite(t *testing.T) {
	nodes := newRaftCluster(t, 3)
	leader, follower := findLeaderFollower(t, nodes)

	step := helpers.NewSimpleStep("forwarded-step")
	w := postJSON(t,
		follower.server.SetupRoutes(), "/engine/step", step, http.StatusCreated,
	)

	var resp api.StepRegisteredResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Step)
	assert.Equal(t, step.ID, resp.Step.ID)

	assert.Eventually(t, func() bool {
		cat, err := leader.engine.GetCatalogState()
		if err != nil {
			return false
		}
		_, ok := cat.Steps[step.ID]
		return ok
	}, 15*time.Second, 100*time.Millisecond)
}

func TestFollowerWriteStartup(t *testing.T) {
	inits := newRaftInits(t, 3)
	type res struct {
		node *raftNode
		err  error
	}

	started := make(chan res, len(inits))
	start := func(init *raftInit) {
		go func() {
			n, err := bootRaftNode(init)
			started <- res{node: n, err: err}
		}()
	}

	start(inits[0])
	time.Sleep(2 * time.Second)
	start(inits[1])
	start(inits[2])

	var nodes []*raftNode
	var wrote bool
	step := newScriptStep()
	for range len(inits) {
		res := <-started
		assert.NoError(t, res.err)

		n := res.node
		nodes = append(nodes, n)
		if !wrote && n.persistence.State() == raft.StateFollower {
			w := postJSON(
				t,
				n.server.SetupRoutes(),
				"/engine/step",
				step,
				http.StatusCreated,
			)

			var resp api.StepRegisteredResponse
			assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.NotNil(t, resp.Step)
			assert.Equal(t, step.ID, resp.Step.ID)
			wrote = true
		}
	}
	assert.True(t, wrote)

	t.Cleanup(func() {
		for _, n := range nodes {
			if n.engine != nil {
				_ = n.engine.Stop()
			}
			if n.flowStore != nil {
				_ = n.flowStore.Close()
			}
			if n.engStore != nil {
				_ = n.engStore.Close()
			}
		}
	})

	leader, _ := findLeaderFollower(t, nodes)
	assert.Eventually(t, func() bool {
		cat, err := leader.engine.GetCatalogState()
		if err != nil {
			return false
		}
		_, ok := cat.Steps[step.ID]
		return ok
	}, 15*time.Second, 100*time.Millisecond)
}

func newRaftCluster(t *testing.T, n int) []*raftNode {
	t.Helper()

	inits := newRaftInits(t, n)
	type res struct {
		node *raftNode
		err  error
	}

	started := make(chan res, len(inits))
	for _, init := range inits {
		go func(init *raftInit) {
			n, err := bootRaftNode(init)
			started <- res{node: n, err: err}
		}(init)
	}

	nodes := make([]*raftNode, 0, len(inits))
	for range len(inits) {
		res := <-started
		assert.NoError(t, res.err)
		nodes = append(nodes, res.node)
	}

	t.Cleanup(func() {
		for _, n := range nodes {
			if n.engine != nil {
				_ = n.engine.Stop()
			}
			if n.flowStore != nil {
				_ = n.flowStore.Close()
			}
			if n.engStore != nil {
				_ = n.engStore.Close()
			}
		}
	})

	return nodes
}

func newRaftInits(t *testing.T, n int) []*raftInit {
	t.Helper()

	srvs := make([]raft.Server, 0, n)
	inits := make([]*raftInit, 0, n)
	for i := range n {
		id := fmt.Sprintf("node-%d", i+1)
		srvs = append(srvs, raft.Server{
			ID:      id,
			Address: freeAddr(t),
		})
		inits = append(inits, &raftInit{id: id})
	}

	for i, init := range inits {
		cfg := helpers.NewTestConfig()
		cfg.APIHost = "127.0.0.1"
		cfg.APIPort = 8080
		cfg.WebhookBaseURL = "http://127.0.0.1"
		cfg.Raft.LocalID = init.id
		cfg.Raft.Address = srvs[i].Address
		cfg.Raft.DataDir = t.TempDir()
		cfg.Raft.Servers = srvs

		hub := event.NewHub()
		cfg.Raft.Publisher = hub.Publish

		init.cfg = cfg
		init.hub = hub
	}
	return inits
}

func bootRaftNode(init *raftInit) (*raftNode, error) {
	p, err := raft.NewPersistence(init.cfg.Raft)
	if err != nil {
		return nil, err
	}

	engStore, err := p.NewStore(init.cfg.EngineStoreConfig())
	if err != nil {
		_ = p.Close()
		return nil, err
	}
	flowStore, err := p.NewStore(init.cfg.FlowStoreConfig())
	if err != nil {
		_ = engStore.Close()
		return nil, err
	}
	closeStore := true
	defer func() {
		if closeStore {
			_ = flowStore.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := flowStore.WaitReady(ctx); err != nil {
		return nil, err
	}

	eng, err := engine.New(init.cfg, engine.Dependencies{
		EngineStore:      engStore,
		FlowStore:        flowStore,
		StepClient:       helpers.NewMockClient(),
		Clock:            time.Now,
		TimerConstructor: scheduler.NewTimer,
		EventHub:         init.hub,
	})
	if err != nil {
		return nil, err
	}
	if err := eng.Start(); err != nil {
		_ = eng.Stop()
		return nil, err
	}

	closeStore = false
	return &raftNode{
		id:          init.id,
		engStore:    engStore,
		flowStore:   flowStore,
		persistence: p,
		engine:      eng,
		server: server.NewServer(
			eng,
			init.hub,
			server.NewRaftStatusProvider(p),
		),
	}, nil
}

func findLeaderFollower(
	t *testing.T, nodes []*raftNode,
) (*raftNode, *raftNode) {
	t.Helper()

	var leader *raftNode
	assert.Eventually(t, func() bool {
		for _, n := range nodes {
			if n.persistence.State() == raft.StateLeader {
				leader = n
				return true
			}
		}
		return false
	}, 15*time.Second, 100*time.Millisecond)

	for _, n := range nodes {
		if n.id != leader.id {
			return leader, n
		}
	}
	t.Fatal("no follower found")
	return nil, nil
}

func postJSON(
	t *testing.T, h http.Handler, path string, body any, want int,
) *httptest.ResponseRecorder {
	t.Helper()

	data, err := json.Marshal(body)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, want, w.Code, w.Body.String())
	return w
}

func freeAddr(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()
	return ln.Addr().String()
}

func newScriptStep() *api.Step {
	return &api.Step{
		ID:   "k6-simple-step",
		Name: "K6 Simple Step",
		Type: api.StepTypeScript,
		Attributes: api.AttributeSpecs{
			"input": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		},
		Script: &api.ScriptConfig{
			Language: "ale",
			Script:   `{:result "hello"}`,
		},
	}
}
