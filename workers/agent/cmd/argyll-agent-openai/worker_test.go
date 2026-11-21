package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/openai"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
	"github.com/kode4food/argyll/workers/agent/internal/workertest"
)

const modelList = `{"data":[
	{"id":"second","shutdown_date":"2099-01-01"},
	{"id":"first"},
	{"id":"retired","shutdown_date":"2000-01-01"}
]}`

func TestWorker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capabilities.json")
	assert.NoError(t, os.WriteFile(path, []byte(`{
		"openai/first":{}, "openai/second":{}, "openai/test-model":{}
	}`), 0o600))
	t.Setenv("MODEL_CAPABILITIES", path)
	models, _, err := openai.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	workertest.Run(t, &workertest.Harness{
		Failures: map[string]string{
			"length": `{"choices":[{"finish_reason":"length","message":{"role":"assistant"}}],
				"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
			"content_filter": `{"choices":[{"finish_reason":"content_filter",
				"message":{"role":"assistant"}}],
				"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
			"refusal": `{"choices":[{"finish_reason":"stop","message":{"role":"assistant",
				"refusal":"refusal explanation"}}],
				"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
		},
		Provider: openai.Name,
		New: func(cfg *worker.Config) worker.Provider {
			cfg.Capabilities = models.Capabilities
			p, err := openai.NewProvider(cfg)
			assert.NoError(t, err)
			return p
		},
		Credentials: credentials,
		ReportsCost: true,
		CacheWrites: true,
		Reply:       reply,
		System: func(body map[string]any) any {
			messages, _ := body["messages"].([]any)
			return messages[0]
		},
		Tools: func(body map[string]any) any {
			return body["tools"]
		},
		ModelList: modelList,
		Models:    []api.AgentModel{{ID: "first"}, {ID: "second"}},
	})
}

func credentials(t *testing.T, set bool) {
	t.Helper()
	key := ""
	if set {
		key = "test"
	}
	t.Setenv("OPENAI_API_KEY", key)
}

func reply(r workertest.Reply) string {
	var calls []map[string]any
	for _, c := range r.Calls {
		calls = append(calls, map[string]any{
			"id":   c.ID,
			"type": "function",
			"function": map[string]any{
				"name":      c.Name,
				"arguments": c.Arguments,
			},
		})
	}
	message := map[string]any{
		"role":       "assistant",
		"content":    nil,
		"tool_calls": calls,
	}
	finish := "tool_calls"
	if len(calls) == 0 {
		message = map[string]any{
			"role":    "assistant",
			"content": "thinking",
		}
		finish = "stop"
	}
	u := r.Usage
	usage := map[string]any{
		"prompt_tokens":     u.Input,
		"completion_tokens": u.Output,
		"total_tokens":      u.Input + u.Output,
		"prompt_tokens_details": map[string]any{
			"cached_tokens":      u.CacheInput,
			"cache_write_tokens": u.CacheOutput,
		},
	}
	if u.Cost != nil {
		usage["cost"] = *u.Cost
	}
	b, _ := json.Marshal(map[string]any{
		"id":      "c",
		"object":  "chat.completion",
		"created": 0,
		"model":   "m",
		"choices": []map[string]any{
			{
				"index":         0,
				"finish_reason": finish,
				"message":       message,
			},
		},
		"usage": usage,
	})
	return string(b)
}
