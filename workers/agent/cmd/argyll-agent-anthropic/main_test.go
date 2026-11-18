package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-anthropic/internal/anthropic"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
	"github.com/kode4food/argyll/workers/agent/internal/workertest"
)

func TestWorker(t *testing.T) {
	workertest.Run(t, anthropic.Name, func(cfg *worker.Config) worker.Provider {
		return anthropic.New(cfg.Providers)
	})
}

func TestDefaultProvider(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test")
	t.Setenv("OPENAI_API_KEY", "test")
	p := anthropic.New(nil)
	assert.Equal(t, []string{anthropic.Name}, p.Names())
	assert.True(t, p.Supports(anthropic.Name))
	assert.False(t, p.Supports("openai"))
}
