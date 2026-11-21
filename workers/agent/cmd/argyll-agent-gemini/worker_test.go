package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-gemini/internal/gemini"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
	"github.com/kode4food/argyll/workers/agent/internal/workertest"
)

const modelList = `{"models":[
	{"name":"models/gemini-second","supportedGenerationMethods":["generateContent"]},
	{"name":"models/gemini-first","supportedGenerationMethods":["generateContent","countTokens"]},
	{"name":"models/gemini-embedding","supportedGenerationMethods":["embedContent"]},
	{"name":"models/gemini-tts","supportedGenerationMethods":["generateContent"]}
]}`

func TestWorker(t *testing.T) {
	testModels(t)
	models, _, err := gemini.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	failures := map[string]string{}
	base := reply(workertest.Reply{Usage: workertest.Usage{
		Input:  10,
		Output: 5,
	}})
	for _, status := range []string{"incomplete", "failed", "budget_exceeded", "cancelled"} {
		failures[status] = strings.Replace(base, `"completed"`, `"`+status+`"`, 1)
	}
	workertest.Run(t, &workertest.Harness{
		Failures: failures,
		Provider: gemini.Name,
		New: func(cfg *worker.Config) worker.Provider {
			cfg.Capabilities = models.Capabilities
			return gemini.NewProvider(cfg)
		},
		Credentials: credentials,
		MissingKey:  "GEMINI_API_KEY",
		Reply:       reply,
		System: func(body map[string]any) any {
			return body["system_instruction"]
		},
		Tools: func(body map[string]any) any {
			return body["tools"]
		},
		ModelList: modelList,
		Models: []api.AgentModel{
			{ID: "gemini-first", Efforts: []api.Effort{"low", "high"}},
			{ID: "gemini-second"},
		},
	})
}

func testModels(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "capabilities.json")
	assert.NoError(t, os.WriteFile(path, []byte(`{
		"gemini/gemini-first":{"efforts":["low","high"]},
		"gemini/gemini-tts":{"hidden":true}
	}`), 0o600))
	t.Setenv("MODEL_CAPABILITIES", path)
}

func credentials(t *testing.T, set bool) {
	t.Helper()
	key := ""
	if set {
		key = "test"
	}
	t.Setenv("GEMINI_API_KEY", key)
}

// reply scripts an interaction. Gemini always sends an arguments object, so other arguments are
// sent as null for the worker to correct
func reply(r workertest.Reply) string {
	var steps []map[string]any
	for _, c := range r.Calls {
		var args any
		if strings.HasPrefix(strings.TrimSpace(c.Arguments), "{") {
			args = json.RawMessage(c.Arguments)
		}
		steps = append(steps, map[string]any{
			"type":      "function_call",
			"id":        c.ID,
			"name":      c.Name,
			"arguments": args,
		})
	}
	status := "requires_action"
	if len(steps) == 0 {
		status = "completed"
		steps = []map[string]any{{
			"type":    "model_output",
			"content": []map[string]any{{"type": "text", "text": "thinking"}},
		}}
	}
	u := r.Usage
	b, _ := json.Marshal(map[string]any{
		"id":     "interaction",
		"status": status,
		"steps":  steps,
		"usage": map[string]any{
			"total_input_tokens":  u.Input,
			"total_output_tokens": u.Output,
			"total_cached_tokens": u.CacheInput,
		},
	})
	return string(b)
}
