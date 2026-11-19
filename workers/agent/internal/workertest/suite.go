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
	suite struct {
		protocol api.ProviderID
		provider func(*worker.Config) worker.Provider
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

	modelToolCall struct {
		id        string
		name      string
		arguments string
	}
)

const (
	providerAnthropic api.ProviderID = "anthropic"
	providerOpenAI    api.ProviderID = "openai"

	crmToken        = "secret"
	anthropicKeyEnv = "ANTHROPIC_API_KEY"
	waitTimeout     = 5 * time.Second
	maxTestTurns    = 5
)

// Run exercises a provider through the shared worker's HTTP contract
func Run(t *testing.T, protocol api.ProviderID, provider func(*worker.Config) worker.Provider) {
	t.Helper()
	s := &suite{
		protocol: protocol,
		provider: provider,
	}
	for _, tc := range []struct {
		name string
		run  func(*testing.T)
	}{
		{name: "invoke", run: s.testInvoke},
		{name: "cost", run: s.testCost},
		{name: "failures", run: s.testInvokeFailures},
		{name: "async", run: s.testInvokeAcceptsBeforeModelAnswers},
		{name: "missing key", run: s.testMissingAPIKey},
		{name: "callback retry", run: s.testCallbackRetry},
		{name: "rejects", run: s.testInvokeRejects},
		{name: "capabilities", run: s.testAgentCapabilities},
		{name: "model discovery failure", run: s.testModelDiscoveryFailure},
		{name: "isolation", run: s.testProviderIsolation},
		{name: "empty capabilities", run: s.testEmptyCapabilities},
		{name: "health", run: s.testHealth},
	} {
		t.Run(tc.name, tc.run)
	}
}

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
	t.Setenv(anthropicKeyEnv, "test")

	t.Run("anthropic with MCP tools", func(t *testing.T) {
		firstReply := anthropicToolUse(modelToolCall{
			id:        "t1",
			name:      "crm__lookup",
			arguments: `{"name": "acme"}`,
		})
		finishReply := anthropicToolUse(modelToolCall{
			id:        "t2",
			name:      "finish",
			arguments: `{"score": 80}`,
		})
		modelUsage := map[string]any{
			"input_tokens":                10,
			"output_tokens":               5,
			"cache_read_input_tokens":     100,
			"cache_creation_input_tokens": 200,
		}
		model := &fakeModel{responses: []string{
			withModelUsage(t, firstReply, modelUsage),
			finishReply,
			finishReply,
			anthropicToolUse(modelToolCall{
				id:        "t3",
				name:      "finish",
				arguments: `{"lead_score": 85}`,
			}),
		}}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()
		crmSrv := crmServer(t)
		defer crmSrv.Close()

		cfg := &worker.Config{
			Servers:   crmConfig(crmSrv.URL),
			Providers: map[api.ProviderID]string{providerAnthropic: modelSrv.URL},
			Prices: map[worker.ModelKey]worker.Price{
				{
					Provider: providerAnthropic,
					Model:    "claude-sonnet-5-5",
				}: {
					Input:       new(float64(3)),
					Output:      new(float64(15)),
					CacheInput:  new(0.3),
					CacheOutput: new(3.75),
				},
			},
		}
		req := agentRequest(providerAnthropic, crmSelection())
		req.Step.Attributes["customer_note"] = &api.AttributeSpec{
			Role: api.RoleOptional,
			Optional: &api.OptionalConfig{
				Mapping: &api.MappingConfig{Name: "note"},
			},
		}
		srv := httptest.NewServer(s.newWorker(t, cfg, providerAnthropic).Handler())
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
		system := mustJSON(t, first["system"])
		assert.Contains(t, system, `\"note\"`)
		assert.NotContains(t, system, "customer_note")
		assert.Contains(t, system, "Score a lead based on company size")
		assert.Contains(t, system, "ephemeral")
		assert.Equal(t, first["system"], second["system"])
		assert.Equal(t, first["tools"], second["tools"])
		assert.Contains(t, mustJSON(t, first["messages"]), "acme")
		assert.Contains(t, mustJSON(t, second), "acme has 40 seats")
		assert.Equal(t, []string{"crm__lookup", "finish"}, toolNames(first))

		req.Step.Name = "Score prospect"
		req.Step.Description = "Score using the updated criteria"
		req.Inputs["lead"] = "other lead"
		out = invokeWorker(t, srv.URL, req)
		assert.Equal(t, api.Args{"score": float64(80)}, out.body.Outputs)
		third := model.request(2)
		system = mustJSON(t, third["system"])
		assert.Contains(t, system, "Step: Score prospect")
		assert.Contains(t, system, req.Step.Description)
		assert.Contains(t, system, `\"note\"`)
		assert.NotContains(t, system, "customer_note")
		assert.Equal(t, first["tools"], third["tools"])
		assert.Contains(t, mustJSON(t, third["messages"]), "other lead")

		// A new definition keeps the Step ID and metadata but uses new attribute mappings
		next := agentRequest(providerAnthropic, crmSelection())
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
		system = mustJSON(t, fourth["system"])
		assert.Contains(t, system, `\"account_note\"`)
		assert.NotContains(t, system, `\"note\"`)
		assert.Contains(t, system, "ephemeral")
		tools := mustJSON(t, fourth["tools"])
		assert.Contains(t, tools, `"lead_score"`)
		assert.NotContains(t, tools, `"score"`)
	})

	t.Run("OpenAI-compatible provider", func(t *testing.T) {
		modelUsage := map[string]any{
			"prompt_tokens":         10,
			"completion_tokens":     5,
			"prompt_tokens_details": map[string]int{"cached_tokens": 4},
		}
		model := &fakeModel{responses: []string{
			withModelUsage(t, openAIText(), modelUsage),
			openAIToolCall(modelToolCall{
				id:        "bad",
				name:      "finish",
				arguments: `{`,
			}),
			openAIToolCall(modelToolCall{
				id:        "missing",
				name:      "unlisted",
				arguments: `{}`,
			}),
			openAIToolCall(modelToolCall{
				id:        "t1",
				name:      "finish",
				arguments: `{"score": 1, "extra": 2}`,
			}),
			openAIToolCall(modelToolCall{
				id:        "t2",
				name:      "finish",
				arguments: `{"score": 7}`,
			}),
		}}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()

		cfg := &worker.Config{
			Providers: map[api.ProviderID]string{"groq": modelSrv.URL},
		}
		req := agentRequest("groq")
		req.Step.Agent.MaxTurns = maxTestTurns
		out := s.invoke(t, cfg, req)

		assert.Equal(t, api.Args{"score": float64(7)}, out.body.Outputs)
		assert.Equal(t, float64(50), out.body.Usage["input_tokens"])
		assert.Equal(t, float64(4), out.body.Usage["cache_input_tokens"])
		assert.Equal(t, "llama-4", model.request(0)["model"])
		assert.Contains(t, mustJSON(t, model.request(1)), "Call the `finish` tool")
		assert.Contains(t, mustJSON(t, model.request(2)), "arguments must be a JSON object")
		assert.Contains(t, mustJSON(t, model.request(3)), "unknown tool unlisted")
		assert.Contains(t, mustJSON(t, model.request(4)), "finish takes exactly score")
	})

	t.Run("corrects a wrong finish", func(t *testing.T) {
		model := &fakeModel{responses: []string{
			anthropicToolUse(modelToolCall{
				id:        "t1",
				name:      "finish",
				arguments: `{"extra": 2}`,
			}),
			anthropicToolUse(modelToolCall{
				id:        "t2",
				name:      "finish",
				arguments: `{"score": 2}`,
			}),
		}}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()

		cfg := &worker.Config{
			Providers: map[api.ProviderID]string{providerAnthropic: modelSrv.URL},
		}
		out := s.invoke(t, cfg, agentRequest(providerAnthropic))

		assert.Equal(t, api.Args{"score": float64(2)}, out.body.Outputs)
		assert.Contains(t, mustJSON(t, model.request(1)), "finish takes exactly score")
	})

	t.Run("reports an empty outputs object", func(t *testing.T) {
		model := &fakeModel{responses: []string{openAIToolCall(modelToolCall{
			id:        "t1",
			name:      "finish",
			arguments: `{}`,
		})}}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()

		req := agentRequest("local")
		delete(req.Step.Attributes, "score")
		cfg := &worker.Config{
			Providers: map[api.ProviderID]string{"local": modelSrv.URL},
		}
		out := s.invoke(t, cfg, req)

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
				Rates: price.Rates,
				Tiers: []worker.PriceTier{{InputOver: 10}},
			},
			usage: api.Usage{"cost": 0.000105, "cost_calculated": 0.000105},
		},
		{
			name: "cached input counts toward the highest qualifying tier",
			price: worker.Price{
				Rates: price.Rates,
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
			finish := openAIToolCall(modelToolCall{
				id:        "finish",
				name:      "finish",
				arguments: `{"score": 80}`,
			})
			reply := finish
			if tc.mixed {
				reply = openAIText()
			}
			usage := map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"cost":              tc.cost,
			}
			if tc.cached {
				usage["prompt_tokens_details"] = map[string]int{
					"cached_tokens":      4,
					"cache_write_tokens": 2,
				}
			}
			model := &fakeModel{responses: []string{withModelUsage(t, reply, usage)}}
			if tc.mixed {
				model.responses = append(model.responses, finish)
			}
			srv := httptest.NewServer(model)
			defer srv.Close()
			cfg := &worker.Config{
				Providers: map[api.ProviderID]string{"local": srv.URL},
				Prices: map[worker.ModelKey]worker.Price{
					{
						Provider: "local",
						Model:    "llama-4",
					}: tc.price,
				},
			}
			out := s.invoke(t, cfg, agentRequest("local"))
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
	t.Setenv(anthropicKeyEnv, "test")

	t.Run("provider status stands", func(t *testing.T) {
		model := &fakeModel{
			status: http.StatusUnauthorized,
			responses: []string{
				`{"type": "error", "error": {"type": "authentication_error", "message": "bad"}}`,
			},
		}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()

		cfg := &worker.Config{
			Providers: map[api.ProviderID]string{providerAnthropic: modelSrv.URL},
		}
		out := s.invoke(t, cfg, agentRequest(providerAnthropic))

		assert.Equal(t, api.JSONContentType, out.contentType)
		assert.Equal(t, http.StatusUnauthorized, out.body.Problem.Status)
		usage := api.Usage{
			"input_tokens":  0,
			"output_tokens": 0,
		}
		assert.Equal(t, usage, out.body.Usage)
	})

	t.Run("turn limit fails permanently", func(t *testing.T) {
		model := &fakeModel{responses: []string{anthropicText(), anthropicText()}}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()

		cfg := &worker.Config{
			Providers: map[api.ProviderID]string{providerAnthropic: modelSrv.URL},
		}
		out := s.invoke(t, cfg, agentRequest(providerAnthropic))

		assert.Equal(t, http.StatusUnprocessableEntity, out.body.Problem.Status)
		assert.Contains(t, mustJSON(t, model.request(1)), "Call the `finish` tool")
	})

	t.Run("OpenAI failures", func(t *testing.T) {
		for _, tc := range []struct {
			name           string
			response       string
			providerStatus int
			wantStatus     int
			usage          api.Usage
		}{
			{
				name: "missing choices",
				response: `{"choices":[],"usage":{"prompt_tokens":10,
					"completion_tokens":2,"cost":0.125}}`,
				providerStatus: http.StatusOK,
				wantStatus:     http.StatusBadGateway,
				usage: api.Usage{
					"input_tokens":  10,
					"output_tokens": 2,
					"cost":          0.125,
					"cost_reported": 0.125,
				},
			},
			{
				name:           "unauthorized",
				response:       `{"error":{"message":"bad key","type":"invalid_api_key"}}`,
				providerStatus: http.StatusUnauthorized,
				wantStatus:     http.StatusUnauthorized,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				model := httptest.NewServer(&fakeModel{
					responses: []string{tc.response},
					status:    tc.providerStatus,
				})
				defer model.Close()
				cfg := &worker.Config{Providers: map[api.ProviderID]string{"local": model.URL}}
				out := s.invoke(t, cfg, agentRequest("local"))
				assert.Equal(t, tc.wantStatus, out.body.Problem.Status)
				if tc.usage != nil {
					assert.Equal(t, tc.usage, out.body.Usage)
				}
			})
		}
	})

	t.Run("unavailable MCP server fails the attempt", func(t *testing.T) {
		crm := httptest.NewServer(http.NotFoundHandler())
		defer crm.Close()
		cfg := &worker.Config{Servers: crmConfig(crm.URL)}
		out := s.invoke(t, cfg, agentRequest(providerAnthropic, crmSelection()))
		assert.Equal(t, http.StatusBadGateway, out.body.Problem.Status)
		assert.Nil(t, out.body.Outputs)
	})

	t.Run("rejects duplicate model tool names", func(t *testing.T) {
		crm := crmServer(t, "ops__lookup")
		defer crm.Close()
		cfg := &worker.Config{Servers: crmConfig(crm.URL)}
		cfg.Servers["crm__ops"] = cfg.Servers["crm"]
		req := agentRequest(providerAnthropic, api.MCPServer{
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

func (s *suite) testInvokeAcceptsBeforeModelAnswers(t *testing.T) {
	t.Setenv(anthropicKeyEnv, "test")
	reply := anthropicToolUse(modelToolCall{
		id:        "t1",
		name:      "finish",
		arguments: `{"score": 1}`,
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
	srv := httptest.NewServer(s.newWorker(t, &worker.Config{
		Providers: map[api.ProviderID]string{providerAnthropic: modelSrv.URL},
	}, providerAnthropic).Handler())
	defer srv.Close()

	resp := postInvoke(t, srv.URL, agentRequest(providerAnthropic), hookSrv.URL)
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
	t.Setenv(anthropicKeyEnv, "")
	model := &fakeModel{}
	modelSrv := httptest.NewServer(model)
	defer modelSrv.Close()

	cfg := &worker.Config{
		Providers: map[api.ProviderID]string{providerAnthropic: modelSrv.URL},
	}
	out := s.invoke(t, cfg, agentRequest(providerAnthropic))

	assert.Equal(t, http.StatusUnauthorized, out.body.Problem.Status)
	assert.Contains(t, out.body.Problem.Detail, anthropicKeyEnv)
	assert.Nil(t, model.request(0))
}

func (s *suite) testCallbackRetry(t *testing.T) {
	reply := openAIToolCall(modelToolCall{
		id:        "t1",
		name:      "finish",
		arguments: `{"score":7}`,
	})
	model := httptest.NewServer(&fakeModel{responses: []string{reply}})
	defer model.Close()
	hook := &webhook{outcomes: make(chan reported, 1)}
	var calls atomic.Int32
	cb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		hook.ServeHTTP(w, r)
	}))
	defer cb.Close()
	cfg := &worker.Config{Providers: map[api.ProviderID]string{"local": model.URL}}
	srv := httptest.NewServer(s.newWorker(t, cfg, providerOpenAI).Handler())
	defer srv.Close()

	resp := postInvoke(t, srv.URL, agentRequest("local"), cb.URL)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	select {
	case out := <-hook.outcomes:
		assert.Equal(t, api.Args{"score": float64(7)}, out.body.Outputs)
		assert.EqualValues(t, 2, calls.Load())
	case <-time.After(waitTimeout):
		assert.Fail(t, "no outcome reported after retry")
	}
}

func (s *suite) testInvokeRejects(t *testing.T) {
	srv := httptest.NewServer(s.newWorker(t, &worker.Config{
		Servers: crmConfig("http://localhost:1"),
	}, providerAnthropic).Handler())
	defer srv.Close()

	valid := agentRequest(providerAnthropic)
	invalid := agentRequest(providerAnthropic)
	invalid.Step.Agent.MaxTurns = 0
	unknownProvider := agentRequest("nowhere")
	unknownServer := agentRequest(providerAnthropic, api.MCPServer{Server: "docs"})
	emptyServer := agentRequest(providerAnthropic, api.MCPServer{})

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
	t.Setenv(anthropicKeyEnv, "test")
	t.Setenv("OPENAI_API_KEY", "")
	var calls atomic.Int32
	models := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", api.JSONContentType)
		switch r.URL.Path {
		case "/v1/models":
			assert.Equal(t, "test", r.Header.Get("x-api-key"))
			_, _ = io.WriteString(w, `{"data":[
				{"id":"claude","lifecycle":"deprecated","capabilities":{
					"effort":{"supported":true,"low":{"supported":true}}
				}},
				{"id":"claude-retired","lifecycle":"retired"}
			],"has_more":false}`)
		case "/openai/models":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"second","shutdown_date":"2099-01-01"},
				{"id":"first"},
				{"id":"retired","shutdown_date":"2000-01-01"}
			]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer models.Close()
	crmSrv := crmServer(t)
	defer crmSrv.Close()
	servers := crmConfig(crmSrv.URL)
	servers["down"] = worker.MCPServer{URL: "http://localhost:1"}
	urls := map[api.ProviderID]string{
		"anthropic": models.URL,
		"local":     models.URL + "/openai/",
	}
	serversList := []api.MCPServer{
		{Server: "crm", Tools: []string{"delete", "lookup"}},
		{Server: "down"},
	}
	providers := []api.AgentProvider{
		{
			Provider: "anthropic",
			Models:   []api.AgentModel{{ID: "claude", Efforts: []api.Effort{"low"}}},
			MCP:      serversList,
		},
		{
			Provider: "local",
			Models:   []api.AgentModel{{ID: "first"}, {ID: "second"}},
			MCP:      serversList,
		},
	}
	for _, provider := range providers {
		t.Run(string(provider.Provider), func(t *testing.T) {
			protocol := providerOpenAI
			if provider.Provider == providerAnthropic {
				protocol = providerAnthropic
			}
			cfg := &worker.Config{
				Servers:   servers,
				Providers: map[api.ProviderID]string{provider.Provider: urls[provider.Provider]},
			}
			handler := s.newWorker(t, cfg, protocol).Handler()
			calls.Store(0)
			for range 2 {
				req := httptest.NewRequest(http.MethodGet, builtins.AgentCapabilitiesPath, nil)
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, req)
				assert.Equal(t, http.StatusOK, resp.Code)
				var list api.AgentCapabilities
				assert.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
				assert.Equal(t, []api.AgentProvider{provider}, list.Providers)
			}
			assert.Equal(t, int32(1), calls.Load())
		})
	}
}

func (s *suite) testModelDiscoveryFailure(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	cfg := &worker.Config{
		Providers: map[api.ProviderID]string{s.protocol: srv.URL},
	}
	handler := s.newWorker(t, cfg, s.protocol).Handler()
	req := httptest.NewRequest(http.MethodGet, builtins.AgentCapabilitiesPath, nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	var res api.AgentCapabilities
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&res))
	assert.Equal(t, []api.AgentProvider{{
		Provider: s.protocol,
		Models:   []api.AgentModel{},
		MCP:      []api.MCPServer{},
	}}, res.Providers)
}

func (s *suite) testProviderIsolation(t *testing.T) {
	for _, tc := range []struct {
		protocol api.ProviderID
		provider api.ProviderID
	}{
		{protocol: providerAnthropic, provider: providerOpenAI},
		{protocol: providerAnthropic, provider: "local"},
		{protocol: providerOpenAI, provider: providerAnthropic},
	} {
		t.Run(string(tc.protocol)+"/"+string(tc.provider), func(t *testing.T) {
			cfg := &worker.Config{
				Providers: map[api.ProviderID]string{
					tc.protocol: "http://localhost:1",
					"local":     "http://localhost:1",
				},
			}
			srv := httptest.NewServer(s.newWorker(t, cfg, tc.protocol).Handler())
			defer srv.Close()
			resp := postInvoke(t, srv.URL, agentRequest(tc.provider), "http://localhost:1")
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
		})
	}
}

func (s *suite) testEmptyCapabilities(t *testing.T) {
	t.Setenv(anthropicKeyEnv, "")
	t.Setenv("OPENAI_API_KEY", "")
	h := s.newWorker(t, &worker.Config{}, s.protocol).Handler()
	req := httptest.NewRequest(http.MethodGet, builtins.AgentCapabilitiesPath, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"providers":[]}`, w.Body.String())
}

func (s *suite) testHealth(t *testing.T) {
	srv := httptest.NewServer(s.newWorker(t, &worker.Config{}, s.protocol).Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/health")
	assert.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func (s *suite) invoke(t *testing.T, cfg *worker.Config, req *api.AgentRequest) reported {
	t.Helper()
	protocol := providerOpenAI
	if req.Step.Agent.Provider == providerAnthropic {
		protocol = providerAnthropic
	}
	srv := httptest.NewServer(s.newWorker(t, cfg, protocol).Handler())
	defer srv.Close()
	return invokeWorker(t, srv.URL, req)
}

func (s *suite) newWorker(
	t *testing.T, cfg *worker.Config, protocol api.ProviderID,
) *worker.Worker {
	t.Helper()
	if protocol != s.protocol {
		t.Skip("exercised by the other provider's test suite")
	}
	return worker.New(cfg, s.provider(cfg))
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
	model := "claude-sonnet-5-5"
	if provider != providerAnthropic {
		model = "llama-4"
	}
	return &api.AgentRequest{
		Step: &api.Step{
			ID:          "score",
			Name:        "Score",
			Description: "Score a lead based on company size",
			Type:        api.StepTypeAgent,
			Agent: &api.AgentConfig{
				Provider:  provider,
				Model:     api.ModelID(model),
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
	mcp.AddTool(server, &mcp.Tool{Name: "lookup", Description: "Look up an account"},
		func(_ context.Context, _ *mcp.CallToolRequest, in lookupInput) (
			*mcp.CallToolResult, any, error,
		) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: in.Name + " has 40 seats"}},
			}, nil, nil
		})
	for _, name := range append([]string{"delete"}, extraTools...) {
		mcp.AddTool(server, &mcp.Tool{Name: name, Description: "Auxiliary tool"},
			func(context.Context, *mcp.CallToolRequest, lookupInput) (
				*mcp.CallToolResult, any, error,
			) {
				return &mcp.CallToolResult{}, nil, nil
			})
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

func anthropicToolUse(call modelToolCall) string {
	return `{"id": "msg", "type": "message", "role": "assistant", "model": "m",
		"content": [{"type": "tool_use", "id": "` + call.id + `",
		"name": "` + call.name + `",
		"input": ` + call.arguments + `}], "stop_reason": "tool_use",
		"usage": {"input_tokens": 10, "output_tokens": 5}}`
}

func anthropicText() string {
	return `{"id": "msg", "type": "message", "role": "assistant", "model": "m",
		"content": [{"type": "text", "text": "thinking"}], "stop_reason": "end_turn",
		"usage": {"input_tokens": 10, "output_tokens": 5}}`
}

func openAIText() string {
	return `{"id":"c","object":"chat.completion","created":0,"model":"m",
		"choices":[{"index":0,"finish_reason":"stop",
		"message":{"role":"assistant","content":"thinking"}}],
		"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
}

func openAIToolCall(call modelToolCall) string {
	quoted, _ := json.Marshal(call.arguments)
	return `{"id": "c", "object": "chat.completion", "created": 0, "model": "m",
		"choices": [{"index": 0, "finish_reason": "tool_calls", "message": {
		"role": "assistant", "content": null, "tool_calls": [{"id": "` + call.id + `",
		"type": "function", "function": {"name": "` + call.name + `", "arguments": ` +
		string(quoted) + `}}]}}],
		"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}}`
}

func withModelUsage(t *testing.T, response string, usage map[string]any) string {
	t.Helper()
	var body map[string]any
	assert.NoError(t, json.Unmarshal([]byte(response), &body))
	body["usage"] = usage
	return mustJSON(t, body)
}

func toolNames(body map[string]any) []string {
	tools, _ := body["tools"].([]any)
	res := make([]string, 0, len(tools))
	for _, t := range tools {
		if m, ok := t.(map[string]any); ok {
			res = append(res, m["name"].(string))
		}
	}
	return res
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	assert.NoError(t, err)
	return strings.TrimSpace(string(data))
}
