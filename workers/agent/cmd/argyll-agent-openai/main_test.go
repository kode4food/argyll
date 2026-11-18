package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-openai/internal/openai"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
	"github.com/kode4food/argyll/workers/agent/internal/workertest"
)

func TestWorker(t *testing.T) {
	workertest.Run(t, openai.Name, func(cfg *worker.Config) worker.Provider {
		return openai.New(cfg.Providers)
	})
}

func TestDefaultProvider(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test")
	t.Setenv("ANTHROPIC_API_KEY", "test")
	p := openai.New(nil)
	assert.Equal(t, []string{openai.Name}, p.Names())
	assert.True(t, p.Supports(openai.Name))
	assert.False(t, p.Supports("anthropic"))
}
