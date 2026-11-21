package workertest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type (
	// Harness adapts one provider implementation to the shared worker suite. The suite serves
	// Reply bodies from a fake model reached through the Provider's base URL
	Harness struct {
		Provider    api.ProviderID
		New         func(*worker.Config) worker.Provider
		Credentials func(t *testing.T, set bool)
		MissingKey  string
		ReportsCost bool
		Reply       func(Reply) string
		System      func(body map[string]any) any
		Tools       func(body map[string]any) any
		ModelList   string
		Models      []api.AgentModel
		Failures    map[string]string
	}

	// Reply is a scripted model response, answering with text when it has no tool calls
	Reply struct {
		Calls []ToolCall
		Usage Usage
	}

	// ToolCall is a tool call in a scripted model response
	ToolCall struct {
		ID        string
		Name      string
		Arguments string
	}

	// Usage is the token accounting a scripted response reports, as the worker counts it
	Usage struct {
		Input       int64
		Output      int64
		CacheInput  int64
		CacheOutput int64
		Cost        *float64
	}

	suite struct {
		h *Harness
	}

	// fakeModel answers each completion with the next scripted response
	fakeModel struct {
		mu        sync.Mutex
		gate      chan struct{}
		status    int
		responses []string
		bodies    []map[string]any
	}

	// webhook captures the outcome a worker reports
	webhook struct {
		outcomes chan reported
	}

	reported struct {
		contentType string
		body        api.AgentOutcome
	}

	lookupInput struct {
		Name string `json:"name"`
	}
)

const (
	testModel             api.ModelID = "test-model"
	crmToken                          = "secret"
	replyInput                        = 10
	replyOutput                       = 5
	waitTimeout                       = 5 * time.Second
	maxTestTurns                      = 5
	permanentReportWindow             = 2 * time.Second
	reportPollInterval                = 10 * time.Millisecond
)

// Run exercises a provider implementation through the shared worker's HTTP contract
func Run(t *testing.T, h *Harness) {
	t.Helper()
	h.Credentials(t, true)
	s := &suite{h: h}
	for _, tc := range []struct {
		name string
		run  func(*testing.T)
	}{
		{name: "invoke", run: s.testInvoke},
		{name: "cost", run: s.testCost},
		{name: "failures", run: s.testInvokeFailures},
		{name: "terminal model responses", run: s.testTerminalResponses},
		{name: "async", run: s.testInvokeAcceptsBeforeModelAnswers},
		{name: "missing key", run: s.testMissingAPIKey},
		{name: "callback retry", run: s.testCallbackRetry},
		{name: "callback rejection", run: s.testCallbackRejection},
		{name: "rejects", run: s.testInvokeRejects},
		{name: "capabilities", run: s.testAgentCapabilities},
		{name: "model discovery failure", run: s.testModelDiscoveryFailure},
		{name: "empty capabilities", run: s.testEmptyCapabilities},
		{name: "health", run: s.testHealth},
	} {
		t.Run(tc.name, tc.run)
	}
}

// ServeHTTP records the request body and answers with the next scripted response
func (m *fakeModel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.gate != nil {
		<-m.gate
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bodies = append(m.bodies, body)
	res := `{}`
	if len(m.responses) > 0 {
		res = m.responses[0]
		m.responses = m.responses[1:]
	}
	w.Header().Set("Content-Type", "application/json")
	if m.status != 0 {
		w.WriteHeader(m.status)
	}
	_, _ = io.WriteString(w, res)
}

// ServeHTTP captures the reported outcome
func (h *webhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body api.AgentOutcome
	_ = json.NewDecoder(r.Body).Decode(&body)
	h.outcomes <- reported{
		contentType: r.Header.Get("Content-Type"),
		body:        body,
	}
	w.WriteHeader(http.StatusOK)
}

func (s *suite) testInvoke(t *testing.T) {
	t.Run("with MCP tools", func(t *testing.T) {
		firstReply := s.h.Reply(Reply{
			Calls: []ToolCall{{ID: "t1", Name: "crm__lookup", Arguments: `{"name": "acme"}`}},
			Usage: Usage{
				Input:       310,
				Output:      replyOutput,
				CacheInput:  100,
				CacheOutput: 200,
			},
		})
		finishReply := s.reply(ToolCall{
			ID:        "t2",
			Name:      "finish",
			Arguments: `{"score": 80}`,
		})
		model := &fakeModel{responses: []string{
			firstReply,
			finishReply,
			finishReply,
			s.reply(ToolCall{
				ID:        "t3",
				Name:      "finish",
				Arguments: `{"lead_score": 85}`,
			}),
		}}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()
		crmSrv := crmServer(t)
		defer crmSrv.Close()

		cfg := s.config(modelSrv.URL)
		cfg.Servers = crmConfig(crmSrv.URL)
		cfg.Prices = map[worker.ModelKey]worker.Price{
			{
				Provider: s.h.Provider,
				Model:    testModel,
			}: {
				Input:       new(float64(3)),
				Output:      new(float64(15)),
				CacheInput:  new(0.3),
				CacheOutput: new(3.75),
			},
		}
		req := s.request(crmSelection())
		req.Step.Attributes["customer_note"] = &api.AttributeSpec{
			Role: api.RoleOptional,
			Optional: &api.OptionalConfig{
				Mapping: &api.MappingConfig{Name: "note"},
			},
		}
		srv := httptest.NewServer(s.newWorker(cfg).Handler())
		defer srv.Close()
		out := invokeWorker(t, srv.URL, req)

		assert.Equal(t, api.JSONContentType, out.contentType)
		assert.Equal(t, api.Args{"score": float64(80)}, out.body.Outputs)
		usage := api.Usage{
			"input_tokens":        320,
			"output_tokens":       10,
			"cache_input_tokens":  100,
			"cache_output_tokens": 200,
			"cost":                0.00099,
			"cost_calculated":     0.00099,
		}
		assert.Equal(t, usage, out.body.Usage)

		first := model.request(0)
		second := model.request(1)
		system := mustJSON(t, s.h.System(first))
		assert.Contains(t, system, `\"note\"`)
		assert.NotContains(t, system, "customer_note")
		assert.Contains(t, system, "Score a lead based on company size")
		assert.Equal(t, s.h.System(first), s.h.System(second))
		assert.NotNil(t, s.h.Tools(first))
		assert.Equal(t, s.h.Tools(first), s.h.Tools(second))
		assert.Contains(t, mustJSON(t, first), "acme")
		assert.Contains(t, mustJSON(t, second), "acme has 40 seats")
		tools := mustJSON(t, s.h.Tools(first))
		assert.Contains(t, tools, `"crm__lookup"`)
		assert.Contains(t, tools, `"finish"`)

		req.Step.Name = "Score prospect"
		req.Step.Description = "Score using the updated criteria"
		req.Inputs["lead"] = "other lead"
		out = invokeWorker(t, srv.URL, req)
		assert.Equal(t, api.Args{"score": float64(80)}, out.body.Outputs)
		third := model.request(2)
		system = mustJSON(t, s.h.System(third))
		assert.Contains(t, system, "Step: Score prospect")
		assert.Contains(t, system, req.Step.Description)
		assert.Contains(t, system, `\"note\"`)
		assert.NotContains(t, system, "customer_note")
		assert.Equal(t, s.h.Tools(first), s.h.Tools(third))
		assert.Contains(t, mustJSON(t, third), "other lead")

		// A new definition keeps the Step ID and metadata but uses new attribute mappings
		next := s.request(crmSelection())
		next.Step.Name = req.Step.Name
		next.Step.Description = req.Step.Description
		next.Step.Attributes["customer_note"] = &api.AttributeSpec{
			Role: api.RoleOptional,
			Optional: &api.OptionalConfig{
				Mapping: &api.MappingConfig{Name: "account_note"},
			},
		}
		next.Step.Attributes["score"].Output = &api.OutputConfig{
			Mapping: &api.MappingConfig{Name: "lead_score"},
		}
		before, err := req.Step.HashKey()
		assert.NoError(t, err)
		after, err := next.Step.HashKey()
		assert.NoError(t, err)
		assert.NotEqual(t, before, after)

		out = invokeWorker(t, srv.URL, next)
		assert.Equal(t, api.Args{"lead_score": float64(85)}, out.body.Outputs)
		fourth := model.request(3)
		system = mustJSON(t, s.h.System(fourth))
		assert.Contains(t, system, `\"account_note\"`)
		assert.NotContains(t, system, `\"note\"`)
		tools = mustJSON(t, s.h.Tools(fourth))
		assert.Contains(t, tools, `"lead_score"`)
		assert.NotContains(t, tools, `"score"`)
	})

	t.Run("corrects bad tool calls", func(t *testing.T) {
		model := &fakeModel{responses: []string{
			s.h.Reply(Reply{
				Usage: Usage{
					Input:      replyInput,
					Output:     replyOutput,
					CacheInput: 4,
				},
			}),
			s.reply(ToolCall{
				ID:        "bad",
				Name:      "finish",
				Arguments: `1`,
			}),
			s.reply(ToolCall{
				ID:        "missing",
				Name:      "unlisted",
				Arguments: `{}`,
			}),
			s.reply(ToolCall{
				ID:        "t1",
				Name:      "finish",
				Arguments: `{"score": 1, "extra": 2}`,
			}),
			s.reply(ToolCall{
				ID:        "t2",
				Name:      "finish",
				Arguments: `{"score": 7}`,
			}),
		}}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()

		req := s.request()
		req.Step.Agent.MaxTurns = maxTestTurns
		out := s.invoke(t, s.config(modelSrv.URL), req)

		assert.Equal(t, api.Args{"score": float64(7)}, out.body.Outputs)
		assert.Equal(t, float64(50), out.body.Usage["input_tokens"])
		assert.Equal(t, float64(4), out.body.Usage["cache_input_tokens"])
		assert.Contains(t, mustJSON(t, model.request(1)), "Call the `finish` tool")
		assert.Contains(t, mustJSON(t, model.request(2)), "arguments must be a JSON object")
		assert.Contains(t, mustJSON(t, model.request(3)), "unknown tool unlisted")
		assert.Contains(t, mustJSON(t, model.request(4)), "finish takes exactly score")
	})

	t.Run("reports an empty outputs object", func(t *testing.T) {
		model := &fakeModel{responses: []string{s.reply(ToolCall{
			ID:        "t1",
			Name:      "finish",
			Arguments: `{}`,
		})}}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()

		req := s.request()
		delete(req.Step.Attributes, "score")
		out := s.invoke(t, s.config(modelSrv.URL), req)

		assert.Equal(t, api.Args{}, out.body.Outputs)
		assert.Nil(t, out.body.Problem)
	})
}

func (s *suite) testCost(t *testing.T) {
	price := worker.Price{
		Input:  new(float64(3)),
		Output: new(float64(15)),
	}
	for _, tc := range []struct {
		name   string
		cost   *float64
		price  worker.Price
		mixed  bool
		cached bool
		usage  api.Usage
	}{
		{
			name:  "provider cost takes precedence",
			cost:  new(0.125),
			price: price,
			usage: api.Usage{"cost": 0.125, "cost_reported": 0.125},
		},
		{
			name:  "provider can report zero",
			cost:  new(float64(0)),
			price: price,
			usage: api.Usage{"cost": 0, "cost_reported": 0},
		},
		{
			name:  "configured rates",
			price: price,
			usage: api.Usage{"cost": 0.000105, "cost_calculated": 0.000105},
		},
		{
			name: "at the threshold uses the base rates",
			price: worker.Price{
				Input:       price.Input,
				Output:      price.Output,
				CacheInput:  price.CacheInput,
				CacheOutput: price.CacheOutput,
				Tiers:       []worker.PriceTier{{InputOver: 10}},
			},
			usage: api.Usage{"cost": 0.000105, "cost_calculated": 0.000105},
		},
		{
			name: "cached input counts toward the highest qualifying tier",
			price: worker.Price{
				Input:       price.Input,
				Output:      price.Output,
				CacheInput:  price.CacheInput,
				CacheOutput: price.CacheOutput,
				Tiers: []worker.PriceTier{
					{
						InputOver:   9,
						Input:       new(float64(6)),
						Output:      new(float64(20)),
						CacheInput:  new(float64(2)),
						CacheOutput: new(float64(8)),
					},
					{
						InputOver: 4,
					},
				},
			},
			cached: true,
			usage: api.Usage{
				"cost":                0.000148,
				"cost_calculated":     0.000148,
				"cache_input_tokens":  4,
				"cache_output_tokens": 2,
			},
		},
		{
			name: "missing rate preserves known charges",
			price: worker.Price{
				Input: new(float64(3)),
			},
			usage: api.Usage{
				"cost":                 0.00003,
				"cost_calculated":      0.00003,
				"cost_unpriced_tokens": 5,
			},
		},
		{
			name:  "unpriced tokens",
			usage: api.Usage{"cost_unpriced_tokens": 15},
		},
		{
			name:   "cache tokens without cache rates",
			price:  price,
			cached: true,
			usage: api.Usage{
				"cost":                 0.000087,
				"cost_calculated":      0.000087,
				"cost_unpriced_tokens": 6,
				"cache_input_tokens":   4,
				"cache_output_tokens":  2,
			},
		},
		{
			name: "configured zero rates",
			price: worker.Price{
				Input:  new(float64(0)),
				Output: new(float64(0)),
			},
			usage: api.Usage{"cost": 0, "cost_calculated": 0},
		},
		{
			name:  "mixed provider and calculated turns",
			cost:  new(0.125),
			price: price,
			mixed: true,
			usage: api.Usage{
				"cost":            0.125105,
				"cost_reported":   0.125,
				"cost_calculated": 0.000105,
			},
		},
		{
			name:  "unpriced turn preserves reported charges",
			cost:  new(0.125),
			mixed: true,
			usage: api.Usage{
				"cost":                 0.125,
				"cost_reported":        0.125,
				"cost_unpriced_tokens": 15,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.cost != nil && !s.h.ReportsCost {
				t.Skip("provider reports no cost")
			}
			finish := ToolCall{
				ID:        "finish",
				Name:      "finish",
				Arguments: `{"score": 80}`,
			}
			usage := Usage{
				Input:  replyInput,
				Output: replyOutput,
				Cost:   tc.cost,
			}
			if tc.cached {
				usage.CacheInput = 4
				usage.CacheOutput = 2
			}
			first := Reply{
				Calls: []ToolCall{finish},
				Usage: usage,
			}
			if tc.mixed {
				first.Calls = nil
			}
			model := &fakeModel{responses: []string{s.h.Reply(first)}}
			if tc.mixed {
				model.responses = append(model.responses, s.reply(finish))
			}
			srv := httptest.NewServer(model)
			defer srv.Close()
			cfg := s.config(srv.URL)
			cfg.Prices = map[worker.ModelKey]worker.Price{
				{
					Provider: s.h.Provider,
					Model:    testModel,
				}: tc.price,
			}
			out := s.invoke(t, cfg, s.request())
			assert.Nil(t, out.body.Problem)
			assert.Equal(t, api.Args{"score": float64(80)}, out.body.Outputs)
			want := tc.usage
			want["input_tokens"] = 10
			want["output_tokens"] = 5
			if tc.mixed {
				want["input_tokens"] = 20
				want["output_tokens"] = 10
			}
			assert.Equal(t, want, out.body.Usage)
		})
	}
}

func (s *suite) testInvokeFailures(t *testing.T) {
	t.Run("provider status stands", func(t *testing.T) {
		model := &fakeModel{status: http.StatusUnauthorized}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()

		out := s.invoke(t, s.config(modelSrv.URL), s.request())

		assert.Equal(t, api.JSONContentType, out.contentType)
		assert.Equal(t, http.StatusUnauthorized, out.body.Problem.Status)
		usage := api.Usage{
			"input_tokens":  0,
			"output_tokens": 0,
		}
		assert.Equal(t, usage, out.body.Usage)
	})

	t.Run("turn limit fails permanently", func(t *testing.T) {
		model := &fakeModel{responses: []string{s.reply(), s.reply()}}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()

		out := s.invoke(t, s.config(modelSrv.URL), s.request())

		assert.Equal(t, http.StatusUnprocessableEntity, out.body.Problem.Status)
		assert.Contains(t, mustJSON(t, model.request(1)), "Call the `finish` tool")
	})

	t.Run("unavailable MCP server fails the attempt", func(t *testing.T) {
		crm := httptest.NewServer(http.NotFoundHandler())
		defer crm.Close()
		cfg := s.config("http://localhost:1")
		cfg.Servers = crmConfig(crm.URL)
		out := s.invoke(t, cfg, s.request(crmSelection()))
		assert.Equal(t, http.StatusBadGateway, out.body.Problem.Status)
		assert.Nil(t, out.body.Outputs)
	})

	t.Run("rejects duplicate model tool names", func(t *testing.T) {
		crm := crmServer(t, "ops__lookup")
		defer crm.Close()
		cfg := s.config("http://localhost:1")
		cfg.Servers = crmConfig(crm.URL)
		cfg.Servers["crm__ops"] = cfg.Servers["crm"]
		req := s.request(api.MCPServer{
			Server: "crm",
			Tools:  []string{"ops__lookup"},
		}, api.MCPServer{
			Server: "crm__ops",
			Tools:  []string{"lookup"},
		})
		out := s.invoke(t, cfg, req)
		assert.Equal(t, http.StatusUnprocessableEntity, out.body.Problem.Status)
		assert.Contains(t, out.body.Problem.Detail, worker.ErrDuplicateTool.Error())
	})
}

func (s *suite) testTerminalResponses(t *testing.T) {
	for reason, body := range s.h.Failures {
		t.Run(reason, func(t *testing.T) {
			model := &fakeModel{responses: []string{body}}
			srv := httptest.NewServer(model)
			defer srv.Close()
			out := s.invoke(t, s.config(srv.URL), s.request())
			if assert.NotNil(t, out.body.Problem) {
				assert.Equal(t, http.StatusUnprocessableEntity, out.body.Problem.Status)
				assert.Contains(t, out.body.Problem.Detail, reason)
				assert.NotContains(t, out.body.Problem.Detail, worker.ErrTurnLimit.Error())
			}
			assert.Equal(t, float64(10), out.body.Usage["input_tokens"])
			assert.Equal(t, float64(5), out.body.Usage["output_tokens"])
			assert.NotNil(t, model.request(0))
			assert.Nil(t, model.request(1))
		})
	}
}

func (s *suite) testInvokeAcceptsBeforeModelAnswers(t *testing.T) {
	reply := s.reply(ToolCall{
		ID:        "t1",
		Name:      "finish",
		Arguments: `{"score": 1}`,
	})
	model := &fakeModel{
		gate:      make(chan struct{}),
		responses: []string{reply},
	}
	modelSrv := httptest.NewServer(model)
	defer modelSrv.Close()
	hook := &webhook{outcomes: make(chan reported, 1)}
	hookSrv := httptest.NewServer(hook)
	defer hookSrv.Close()
	srv := httptest.NewServer(s.newWorker(s.config(modelSrv.URL)).Handler())
	defer srv.Close()

	resp := postInvoke(t, srv.URL, s.request(), hookSrv.URL)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	assert.Nil(t, model.request(0))

	close(model.gate)
	select {
	case out := <-hook.outcomes:
		assert.Equal(t, api.Args{"score": float64(1)}, out.body.Outputs)
	case <-time.After(waitTimeout):
		assert.Fail(t, "no outcome reported")
	}
}

func (s *suite) testMissingAPIKey(t *testing.T) {
	if s.h.MissingKey == "" {
		t.Skip("provider runs without credentials")
	}
	s.h.Credentials(t, false)
	model := &fakeModel{}
	modelSrv := httptest.NewServer(model)
	defer modelSrv.Close()

	out := s.invoke(t, s.config(modelSrv.URL), s.request())

	assert.Equal(t, http.StatusUnauthorized, out.body.Problem.Status)
	assert.Contains(t, out.body.Problem.Detail, s.h.MissingKey)
	assert.Nil(t, model.request(0))
}

func (s *suite) testCallbackRetry(t *testing.T) {
	for _, status := range []int{
		http.StatusServiceUnavailable, http.StatusRequestTimeout, http.StatusTooManyRequests,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			reply := s.reply(ToolCall{
				ID:        "t1",
				Name:      "finish",
				Arguments: `{"score":7}`,
			})
			model := httptest.NewServer(&fakeModel{responses: []string{reply}})
			defer model.Close()
			hook := &webhook{outcomes: make(chan reported, 1)}
			var calls atomic.Int32
			cb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					w.WriteHeader(status)
					return
				}
				hook.ServeHTTP(w, r)
			}))
			defer cb.Close()
			srv := httptest.NewServer(s.newWorker(s.config(model.URL)).Handler())
			defer srv.Close()
			resp := postInvoke(t, srv.URL, s.request(), cb.URL)
			_ = resp.Body.Close()
			assert.Equal(t, http.StatusAccepted, resp.StatusCode)
			select {
			case out := <-hook.outcomes:
				assert.Equal(t, api.Args{"score": float64(7)}, out.body.Outputs)
				assert.EqualValues(t, 2, calls.Load())
			case <-time.After(waitTimeout):
				assert.Fail(t, "no outcome reported after retry")
			}
		})
	}
}

func (s *suite) testCallbackRejection(t *testing.T) {
	reply := s.reply(ToolCall{
		ID:        "t1",
		Name:      "finish",
		Arguments: `{"score":7}`,
	})
	model := httptest.NewServer(&fakeModel{responses: []string{reply}})
	defer model.Close()
	calls := make(chan struct{}, 2)
	cb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", api.ProblemJSONContentType)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(api.NewProblem(http.StatusBadRequest, "invalid receipt"))
		calls <- struct{}{}
	}))
	defer cb.Close()
	srv := httptest.NewServer(s.newWorker(s.config(model.URL)).Handler())
	defer srv.Close()
	resp := postInvoke(t, srv.URL, s.request(), cb.URL)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	select {
	case <-calls:
	case <-time.After(waitTimeout):
		assert.Fail(t, "no outcome reported")
		return
	}
	assert.Never(t,
		func() bool { return len(calls) > 0 }, permanentReportWindow, reportPollInterval)
}

func (s *suite) testInvokeRejects(t *testing.T) {
	cfg := s.config("http://localhost:1")
	cfg.Servers = crmConfig("http://localhost:1")
	srv := httptest.NewServer(s.newWorker(cfg).Handler())
	defer srv.Close()

	valid := s.request()
	invalid := s.request()
	invalid.Step.Agent.MaxTurns = 0
	unknownProvider := agentRequest("nowhere")
	unknownServer := s.request(api.MCPServer{Server: "docs"})
	emptyServer := s.request(api.MCPServer{})

	for _, tc := range []struct {
		name    string
		body    any
		webhook string
		status  int
	}{
		{
			name:   "missing webhook URL",
			body:   valid,
			status: http.StatusBadRequest,
		},
		{
			name:    "invalid request",
			body:    invalid,
			webhook: "http://engine",
			status:  http.StatusBadRequest,
		},
		{
			name:    "malformed request",
			body:    "{",
			webhook: "http://engine",
			status:  http.StatusBadRequest,
		},
		{
			name:    "unknown provider",
			body:    unknownProvider,
			webhook: "http://engine",
			status:  http.StatusUnprocessableEntity,
		},
		{
			name:    "unknown MCP server",
			body:    unknownServer,
			webhook: "http://engine",
			status:  http.StatusUnprocessableEntity,
		},
		{
			name:    "missing provider",
			body:    agentRequest(""),
			webhook: "http://engine",
			status:  http.StatusBadRequest,
		},
		{
			name:    "missing MCP server name",
			body:    emptyServer,
			webhook: "http://engine",
			status:  http.StatusBadRequest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := postInvoke(t, srv.URL, tc.body, tc.webhook)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, tc.status, resp.StatusCode)
			assert.Equal(t, api.ProblemJSONContentType, resp.Header.Get("Content-Type"))
		})
	}
}

func (s *suite) testAgentCapabilities(t *testing.T) {
	var calls atomic.Int32
	models := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", api.JSONContentType)
		_, _ = io.WriteString(w, s.h.ModelList)
	}))
	defer models.Close()
	crmSrv := crmServer(t)
	defer crmSrv.Close()
	cfg := s.config(models.URL)
	cfg.Servers = crmConfig(crmSrv.URL)
	cfg.Servers["down"] = worker.MCPServer{URL: "http://localhost:1"}
	want := []api.AgentProvider{
		{
			Provider: s.h.Provider,
			Models:   s.h.Models,
			MCP: []api.MCPServer{
				{Server: "crm", Tools: []string{"delete", "lookup"}},
				{Server: "down"},
			},
		},
	}
	handler := s.newWorker(cfg).Handler()
	listed := int32(0)
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, builtins.AgentCapabilitiesPath, nil)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
		var list api.AgentCapabilities
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
		assert.Equal(t, want, list.Providers)
		if assert.Len(t, list.Problems, 1) {
			assert.Equal(t, "mcp/down", list.Problems[0].Instance)
			assert.NotEmpty(t, list.Problems[0].Detail)
		}
		if listed == 0 {
			listed = calls.Load()
		}
	}
	// The second request is served from the cached listing
	assert.Positive(t, listed)
	assert.Equal(t, listed, calls.Load())
}

func (s *suite) testModelDiscoveryFailure(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	handler := s.newWorker(s.config(srv.URL)).Handler()
	req := httptest.NewRequest(http.MethodGet, builtins.AgentCapabilitiesPath, nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	var res api.AgentCapabilities
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&res))
	assert.Equal(t, []api.AgentProvider{{
		Provider: s.h.Provider,
		Models:   []api.AgentModel{},
		MCP:      []api.MCPServer{},
	}}, res.Providers)
	if assert.Len(t, res.Problems, 1) {
		assert.Equal(t, "provider/"+string(s.h.Provider), res.Problems[0].Instance)
		assert.Equal(t, http.StatusNotFound, res.Problems[0].Status)
		assert.NotEmpty(t, res.Problems[0].Detail)
	}
}

func (s *suite) testEmptyCapabilities(t *testing.T) {
	s.h.Credentials(t, false)
	h := s.newWorker(&worker.Config{}).Handler()
	req := httptest.NewRequest(http.MethodGet, builtins.AgentCapabilitiesPath, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"providers":[]}`, w.Body.String())
}

func (s *suite) testHealth(t *testing.T) {
	srv := httptest.NewServer(s.newWorker(&worker.Config{}).Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/health")
	assert.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func (s *suite) invoke(t *testing.T, cfg *worker.Config, req *api.AgentRequest) reported {
	t.Helper()
	srv := httptest.NewServer(s.newWorker(cfg).Handler())
	defer srv.Close()
	return invokeWorker(t, srv.URL, req)
}

func (s *suite) newWorker(cfg *worker.Config) *worker.Worker {
	return worker.New(cfg, s.h.New(cfg))
}

func (s *suite) config(url string) *worker.Config {
	return &worker.Config{
		Providers: map[api.ProviderID]string{s.h.Provider: url},
	}
}

func (s *suite) request(mcpServers ...api.MCPServer) *api.AgentRequest {
	return agentRequest(s.h.Provider, mcpServers...)
}

// reply scripts a response with the usual token counts
func (s *suite) reply(calls ...ToolCall) string {
	return s.h.Reply(Reply{
		Calls: calls,
		Usage: Usage{
			Input:  replyInput,
			Output: replyOutput,
		},
	})
}

func (m *fakeModel) request(i int) map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	if i >= len(m.bodies) {
		return nil
	}
	return m.bodies[i]
}

func invokeWorker(t *testing.T, url string, req *api.AgentRequest) reported {
	t.Helper()
	hook := &webhook{outcomes: make(chan reported, 1)}
	hookSrv := httptest.NewServer(hook)
	defer hookSrv.Close()

	resp := postInvoke(t, url, req, hookSrv.URL)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	select {
	case out := <-hook.outcomes:
		return out
	case <-time.After(waitTimeout):
		assert.Fail(t, "no outcome reported")
		return reported{}
	}
}

func postInvoke(t *testing.T, base string, body any, webhook string) *http.Response {
	t.Helper()
	data, ok := body.(string)
	if !ok {
		data = mustJSON(t, body)
	}
	req, err := http.NewRequest(http.MethodPost, base+"/invoke", bytes.NewBufferString(data))
	assert.NoError(t, err)
	if webhook != "" {
		req.Header.Set(api.HeaderWebhookURL, webhook)
	}
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	return resp
}

func agentRequest(provider api.ProviderID, mcpServers ...api.MCPServer) *api.AgentRequest {
	return &api.AgentRequest{
		Step: &api.Step{
			ID:          "score",
			Name:        "Score",
			Description: "Score a lead based on company size",
			Type:        api.StepTypeAgent,
			Agent: &api.AgentConfig{
				Provider:  provider,
				Model:     testModel,
				Prologue:  "# Score",
				MCP:       mcpServers,
				MaxTurns:  2,
				MaxTokens: 100,
			},
			Attributes: api.AttributeSpecs{
				"lead":  {Role: api.RoleRequired},
				"score": {Role: api.RoleOutput},
			},
		},
		Inputs:  api.Args{"lead": "acme"},
		Timeout: waitTimeout.Milliseconds(),
	}
}

func crmSelection() api.MCPServer {
	return api.MCPServer{
		Server: "crm",
		Tools:  []string{"lookup"},
	}
}

func crmConfig(url string) map[string]worker.MCPServer {
	return map[string]worker.MCPServer{
		"crm": {
			URL:     url,
			Headers: map[string]string{"Authorization": "Bearer " + crmToken},
		},
	}
}

// crmServer serves two MCP tools, refusing requests without the configured token
func crmServer(t *testing.T, extraTools ...string) *httptest.Server {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "crm",
		Version: "1",
	}, nil)
	mcp.AddTool(
		server, &mcp.Tool{
			Name:        "lookup",
			Description: "Look up an account",
		},
		func(_ context.Context, _ *mcp.CallToolRequest, in lookupInput) (
			*mcp.CallToolResult, any, error,
		) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: in.Name + " has 40 seats"}},
			}, nil, nil
		},
	)
	for _, name := range append([]string{"delete"}, extraTools...) {
		mcp.AddTool(
			server, &mcp.Tool{
				Name:        name,
				Description: "Auxiliary tool",
			},
			func(context.Context, *mcp.CallToolRequest, lookupInput) (
				*mcp.CallToolResult, any, error,
			) {
				return &mcp.CallToolResult{}, nil, nil
			},
		)
	}
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+crmToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	assert.NoError(t, err)
	return strings.TrimSpace(string(data))
}
