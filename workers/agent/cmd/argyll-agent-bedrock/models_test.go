package main_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-bedrock/internal/bedrock"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

const modelCapabilities = `{
	"bedrock/anthropic.claude-test":{"efforts":["low","high"]},
	"bedrock/amazon.nova-test":{"efforts":["low","high"]},
	"bedrock/openai.test":{"efforts":["low","high"]},
	"bedrock/other.test":{"efforts":["high"]},
	"bedrock/blocked":{"hidden":true}
}`

func TestEffort(t *testing.T) {
	for _, tc := range []struct {
		model  api.ModelID
		effort api.Effort
		fields map[string]any
		err    error
	}{
		{
			model:  "anthropic.claude-test",
			effort: "low",
			fields: map[string]any{
				"output_config": map[string]any{"effort": "low"},
			},
		},
		{
			model:  "amazon.nova-test",
			effort: "high",
			fields: map[string]any{"reasoningConfig": map[string]any{
				"type": "enabled", "maxReasoningEffort": "high",
			}},
		},
		{
			model:  "openai.test",
			effort: "high",
			fields: map[string]any{"reasoning_effort": "high"},
		},
		{
			model: "anthropic.claude-test",
		},
		{
			model:  "anthropic.claude-test",
			effort: "max",
			err:    bedrock.ErrInvalidEffort,
		},
		{
			model:  "other.test",
			effort: "high",
			err:    bedrock.ErrInvalidEffort,
		},
		{
			model: "blocked",
			err:   bedrock.ErrUnsupportedModel,
		},
	} {
		t.Run(string(tc.model)+"/"+string(tc.effort), func(t *testing.T) {
			credentials(t, true)
			capabilities(t, modelCapabilities)
			bodies := make(chan map[string]any, 2)
			handler := func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				bodies <- body
				w.Header().Set("Content-Type", api.JSONContentType)
				_, _ = io.WriteString(w, `{"output":{"message":{"role":"assistant",
					"content":[{"text":"ok"}]}},"stopReason":"end_turn"}`)
			}
			srv := httptest.NewServer(http.HandlerFunc(handler))
			defer srv.Close()
			t.Setenv("AWS_ENDPOINT_URL", srv.URL)
			_, p, err := bedrock.LoadConfig()
			if !assert.NoError(t, err) {
				return
			}
			c, err := p.Open(&worker.Chat{Config: &api.AgentConfig{
				Model:     tc.model,
				Effort:    tc.effort,
				MaxTokens: 16,
			}})
			assert.ErrorIs(t, err, tc.err)
			if tc.err != nil {
				assert.Empty(t, bodies)
				problem, ok := errors.AsType[*api.ProblemDetails](err)
				if assert.True(t, ok) {
					assert.Equal(t, http.StatusUnprocessableEntity, problem.Status)
				}
				return
			}
			if !assert.NoError(t, err) {
				return
			}
			_, err = c.Turn(t.Context())
			assert.NoError(t, err)
			c.Nudge("continue")
			_, err = c.Turn(t.Context())
			assert.NoError(t, err)
			if !assert.Len(t, bodies, 2) {
				return
			}
			first := <-bodies
			second := <-bodies
			if tc.fields == nil {
				assert.Nil(t, first["additionalModelRequestFields"])
				assert.Nil(t, second["additionalModelRequestFields"])
				return
			}
			assert.Equal(t, tc.fields, first["additionalModelRequestFields"])
			assert.Equal(t, tc.fields, second["additionalModelRequestFields"])
		})
	}
}

func TestProfiles(t *testing.T) {
	credentials(t, true)
	capabilities(t, modelCapabilities)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", api.JSONContentType)
		if r.URL.Path == "/foundation-models" {
			_, _ = io.WriteString(w, `{"modelSummaries":[
				{"modelId":"anthropic.claude-test","inferenceTypesSupported":["ON_DEMAND"],
				 "outputModalities":["TEXT"]},
				{"modelId":"blocked","inferenceTypesSupported":["ON_DEMAND"]},
				{"modelId":"unknown","inferenceTypesSupported":["ON_DEMAND"]},
				{"modelId":"new.text","inferenceTypesSupported":["ON_DEMAND"],
				 "outputModalities":["TEXT"]},
				{"modelId":"openai.test","inferenceTypesSupported":["ON_DEMAND"],
				 "outputModalities":["TEXT","IMAGE"]},
				{"modelId":"amazon.nova-test","inferenceTypesSupported":["INFERENCE_PROFILE"],
				"modelLifecycle":{"endOfLifeTime":"2000-01-01T00:00:00Z"}}
			]}`)
			return
		}
		_, _ = io.WriteString(w, `{"inferenceProfileSummaries":[
			{"inferenceProfileId":"valid","status":"ACTIVE","models":[
				{"modelArn":"arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-test"},
				{"modelArn":"arn:aws:bedrock:us-west-2::foundation-model/anthropic.claude-test"}]},
			{"inferenceProfileId":"blocked","status":"ACTIVE","models":[
				{"modelArn":"arn:aws:bedrock:us-east-1::foundation-model/blocked"}]},
			{"inferenceProfileId":"retired","status":"ACTIVE","models":[
				{"modelArn":"arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-test"}]},
			{"inferenceProfileId":"multimodal-output","status":"ACTIVE","models":[
				{"modelArn":"arn:aws:bedrock:us-east-1::foundation-model/openai.test"}]},
			{"inferenceProfileId":"inactive","status":"INACTIVE"},
			{"inferenceProfileId":"empty","status":"ACTIVE"},
			{"inferenceProfileId":"malformed","status":"ACTIVE","models":[{"modelArn":"bad"}]},
			{"inferenceProfileId":"mixed","status":"ACTIVE","models":[
				{"modelArn":"arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-test"},
				{"modelArn":"arn:aws:bedrock:us-east-1::foundation-model/openai.test"}]}
		]}`)
	}))
	defer srv.Close()
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)
	_, p, err := bedrock.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	models, err := p.Models(t.Context(), bedrock.Name)
	assert.NoError(t, err)
	assert.Equal(t, []api.AgentModel{
		{ID: "anthropic.claude-test", Efforts: []api.Effort{"low", "high"}},
		{ID: "new.text"},
		{ID: "valid", Efforts: []api.Effort{"low", "high"}},
	}, models)
	// A text model the capability table doesn't mention is usable without a refresh
	_, err = p.Open(&worker.Chat{Config: &api.AgentConfig{Model: "new.text"}})
	assert.NoError(t, err)
}

func TestProfileEffort(t *testing.T) {
	credentials(t, true)
	capabilities(t, modelCapabilities)
	calls := make(chan string, 16)
	var removed atomic.Bool
	var failed atomic.Bool
	profiles := `{"inferenceProfileSummaries":[{
		"inferenceProfileId":"application-profile",
		"inferenceProfileArn":
		"arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/app",
		"status":"ACTIVE",
		"models":[{"modelArn":
		"arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-test"}]
	}]}`
	handler := func(w http.ResponseWriter, r *http.Request) {
		calls <- r.URL.Path
		w.Header().Set("Content-Type", api.JSONContentType)
		switch r.URL.Path {
		case "/foundation-models":
			_, _ = io.WriteString(w, `{"modelSummaries":[]}`)
		case "/inference-profiles":
			if failed.Load() {
				http.NotFound(w, r)
				return
			}
			if removed.Load() {
				_, _ = io.WriteString(w, `{"inferenceProfileSummaries":[]}`)
				return
			}
			_, _ = io.WriteString(w, profiles)
		case "/model/application-profile/converse":
			var body map[string]any
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, map[string]any{
				"output_config": map[string]any{"effort": "high"},
			}, body["additionalModelRequestFields"])
			_, _ = io.WriteString(w, `{"output":{"message":{"role":"assistant",
				"content":[{"text":"continue"}]}}}`)
		default:
			http.NotFound(w, r)
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)
	_, p, err := bedrock.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	_, err = p.Open(&worker.Chat{Config: &api.AgentConfig{Model: "application-profile"}})
	assert.ErrorIs(t, err, bedrock.ErrProfilesUnavailable)
	problem, ok := errors.AsType[*api.ProblemDetails](err)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusServiceUnavailable, problem.Status)
	}
	assert.Empty(t, calls)

	_, err = p.Models(t.Context(), bedrock.Name)
	assert.NoError(t, err)
	if !assert.Len(t, calls, 2) {
		return
	}
	assert.Equal(t, "/foundation-models", <-calls)
	assert.Equal(t, "/inference-profiles", <-calls)
	assert.Empty(t, calls)

	for i := range 2 {
		c, err := p.Open(&worker.Chat{Config: &api.AgentConfig{
			Model:  "application-profile",
			Effort: "high",
		}})
		if !assert.NoError(t, err) {
			return
		}
		assert.Len(t, calls, i*2)
		_, err = c.Turn(t.Context())
		assert.NoError(t, err)
		c.Nudge("continue")
		_, err = c.Turn(t.Context())
		assert.NoError(t, err)
	}
	if !assert.Len(t, calls, 4) {
		return
	}
	for range 4 {
		assert.Equal(t, "/model/application-profile/converse", <-calls)
	}
	assert.Empty(t, calls)

	failed.Store(true)
	_, err = p.Models(t.Context(), bedrock.Name)
	assert.Error(t, err)
	if !assert.Len(t, calls, 2) {
		return
	}
	<-calls
	<-calls
	_, err = p.Open(&worker.Chat{Config: &api.AgentConfig{Model: "application-profile"}})
	assert.NoError(t, err)
	assert.Empty(t, calls)
	failed.Store(false)

	removed.Store(true)
	_, err = p.Models(t.Context(), bedrock.Name)
	assert.NoError(t, err)
	if !assert.Len(t, calls, 2) {
		return
	}
	<-calls
	<-calls
	// An unlisted ID is passed through, so Bedrock decides whether it exists
	_, err = p.Open(&worker.Chat{Config: &api.AgentConfig{Model: "application-profile"}})
	assert.NoError(t, err)
	assert.Empty(t, calls)
}

func capabilities(t *testing.T, data string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "capabilities.json")
	assert.NoError(t, os.WriteFile(path, []byte(data), 0o600))
	t.Setenv("MODEL_CAPABILITIES", path)
}
