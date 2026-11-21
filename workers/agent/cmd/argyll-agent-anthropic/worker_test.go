package main_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/anthropic"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
	"github.com/kode4food/argyll/workers/agent/internal/workertest"
)

const modelList = `{"data":[
	{"id":"claude","lifecycle":"deprecated","capabilities":{
		"effort":{"supported":true,"low":{"supported":true}}
	}},
	{"id":"claude-retired","lifecycle":"retired"}
],"has_more":false}`

func TestWorker(t *testing.T) {
	t.Setenv("PREFIX_THRESHOLDS", "")
	failures := map[string]string{}
	base := reply(workertest.Reply{Usage: workertest.Usage{
		Input:  10,
		Output: 5,
	}})
	for _, reason := range []string{"max_tokens", "refusal", "model_context_window_exceeded"} {
		failures[reason] = strings.Replace(base, `"end_turn"`, `"`+reason+`"`, 1)
	}
	workertest.Run(t, &workertest.Harness{
		Failures: failures,
		Provider: anthropic.Name,
		New: func(cfg *worker.Config) worker.Provider {
			p, err := anthropic.NewProvider(cfg.Providers)
			assert.NoError(t, err)
			return p
		},
		Credentials: credentials,
		MissingKey:  "ANTHROPIC_API_KEY",
		Reply:       reply,
		System: func(body map[string]any) any {
			return body["system"]
		},
		Tools: func(body map[string]any) any {
			return body["tools"]
		},
		ModelList: modelList,
		Models: []api.AgentModel{
			{ID: "claude", Efforts: []api.Effort{"low"}},
		},
	})
}

func credentials(t *testing.T, set bool) {
	t.Helper()
	key := ""
	if set {
		key = "test"
	}
	t.Setenv("ANTHROPIC_API_KEY", key)
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
}

func reply(r workertest.Reply) string {
	var content []map[string]any
	for _, c := range r.Calls {
		content = append(content, map[string]any{
			"type":  "tool_use",
			"id":    c.ID,
			"name":  c.Name,
			"input": json.RawMessage(c.Arguments),
		})
	}
	stop := "tool_use"
	if len(content) == 0 {
		content = []map[string]any{{"type": "text", "text": "thinking"}}
		stop = "end_turn"
	}
	u := r.Usage
	b, _ := json.Marshal(map[string]any{
		"id":          "msg",
		"type":        "message",
		"role":        "assistant",
		"model":       "m",
		"content":     content,
		"stop_reason": stop,
		"usage": map[string]any{
			"input_tokens":                u.Input - u.CacheInput - u.CacheOutput,
			"output_tokens":               u.Output,
			"cache_read_input_tokens":     u.CacheInput,
			"cache_creation_input_tokens": u.CacheOutput,
		},
	})
	return string(b)
}
