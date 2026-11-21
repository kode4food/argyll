package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-bedrock/internal/bedrock"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
	"github.com/kode4food/argyll/workers/agent/internal/workertest"
)

// modelList answers both listing calls, since each reads only its own field
const modelList = `{
	"modelSummaries": [
		{"modelId": "amazon.nova-pro-v1:0", "inferenceTypesSupported":["ON_DEMAND"],
		 "outputModalities":["TEXT"]},
		{"modelId": "meta.llama3-70b-instruct-v1:0", "inferenceTypesSupported":["ON_DEMAND"]},
		{"modelId": "amazon.titan-retired", "modelLifecycle": {
			"status": "LEGACY", "endOfLifeTime": "2000-01-01T00:00:00Z"
		}}
	],
	"inferenceProfileSummaries": [
		{"inferenceProfileId": "us.anthropic.claude-sonnet", "status": "ACTIVE", "models":[{
			"modelArn":"arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-sonnet"
		}]}
	]
}`

func TestWorker(t *testing.T) {
	testModels(t)
	models, _, err := bedrock.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	failures := map[string]string{}
	base := reply(workertest.Reply{Usage: workertest.Usage{
		Input:  10,
		Output: 5,
	}})
	for _, reason := range []string{
		"max_tokens", "guardrail_intervened", "content_filtered", "model_context_window_exceeded",
		"malformed_model_output", "malformed_tool_use",
	} {
		failures[reason] = strings.Replace(base, `"end_turn"`, `"`+reason+`"`, 1)
	}
	workertest.Run(t, &workertest.Harness{
		Failures: failures,
		Provider: bedrock.Name,
		New: func(cfg *worker.Config) worker.Provider {
			// The worker reaches the suite's fake model through the AWS SDK's own endpoint
			t.Setenv("AWS_ENDPOINT_URL", cfg.Providers[bedrock.Name])
			cfg.Capabilities = models.Capabilities
			p, err := bedrock.NewProvider(cfg)
			assert.NoError(t, err)
			return p
		},
		Credentials: credentials,
		MissingKey:  "AWS_REGION",
		CacheWrites: true,
		Reply:       reply,
		System: func(body map[string]any) any {
			return body["system"]
		},
		Tools: func(body map[string]any) any {
			return body["toolConfig"]
		},
		ModelList: modelList,
		Models: []api.AgentModel{
			{ID: "amazon.nova-pro-v1:0"},
			{ID: "us.anthropic.claude-sonnet"},
		},
	})
}

func testModels(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "capabilities.json")
	assert.NoError(t, os.WriteFile(path, []byte(`{
		"bedrock/test-model":{}, "bedrock/m":{},
		"bedrock/anthropic.claude-sonnet":{},
		"bedrock/amazon.nova-pro-v1:0":{},
		"bedrock/meta.llama3-70b-instruct-v1:0":{"hidden":true}
	}`), 0o600))
	t.Setenv("MODEL_CAPABILITIES", path)
}

// credentials isolates the worker from the machine's AWS configuration
func credentials(t *testing.T, set bool) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	for _, key := range []string{
		"AWS_PROFILE", "AWS_DEFAULT_REGION", "AWS_SESSION_TOKEN", "AWS_ENDPOINT_URL",
	} {
		t.Setenv(key, "")
	}
	region := ""
	keyID := ""
	secret := ""
	if set {
		region = "us-east-1"
		keyID = "AKIDTEST"
		secret = "secret"
	}
	t.Setenv("AWS_REGION", region)
	t.Setenv("AWS_ACCESS_KEY_ID", keyID)
	t.Setenv("AWS_SECRET_ACCESS_KEY", secret)
}

func reply(r workertest.Reply) string {
	var content []map[string]any
	for _, c := range r.Calls {
		content = append(content, map[string]any{
			"toolUse": map[string]any{
				"toolUseId": c.ID,
				"name":      c.Name,
				"input":     json.RawMessage(c.Arguments),
			},
		})
	}
	stop := "tool_use"
	if len(content) == 0 {
		content = []map[string]any{{"text": "thinking"}}
		stop = "end_turn"
	}
	u := r.Usage
	input := u.Input - u.CacheInput - u.CacheOutput
	b, _ := json.Marshal(map[string]any{
		"output": map[string]any{
			"message": map[string]any{
				"role":    "assistant",
				"content": content,
			},
		},
		"stopReason": stop,
		"usage": map[string]any{
			"inputTokens":           input,
			"outputTokens":          u.Output,
			"totalTokens":           u.Input + u.Output,
			"cacheReadInputTokens":  u.CacheInput,
			"cacheWriteInputTokens": u.CacheOutput,
		},
		"metrics": map[string]any{"latencyMs": 1},
	})
	return string(b)
}
