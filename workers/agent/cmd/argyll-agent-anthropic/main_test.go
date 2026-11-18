package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-anthropic/internal/anthropic"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
	"github.com/kode4food/argyll/workers/agent/internal/workertest"
)

func TestWorker(t *testing.T) {
	t.Setenv("PREFIX_THRESHOLDS", "")
	workertest.Run(t, anthropic.Name, func(cfg *worker.Config) worker.Provider {
		p, err := anthropic.New(cfg.Providers)
		assert.NoError(t, err)
		return p
	})
}

func TestDefaultProvider(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test")
	t.Setenv("OPENAI_API_KEY", "test")
	t.Setenv("PREFIX_THRESHOLDS", "")
	p, err := anthropic.New(nil)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []string{anthropic.Name}, p.Names())
	assert.True(t, p.Supports(anthropic.Name))
	assert.False(t, p.Supports("openai"))
}
