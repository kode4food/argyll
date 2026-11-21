package main_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kode4food/argyll/workers/agent/internal/openai"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	isolateConfig(t)
	t.Setenv("PROVIDER_URLS", "invalid")
	_, _, err := openai.LoadConfig()
	assert.ErrorIs(t, err, worker.ErrInvalidProviders)
	t.Setenv("PROVIDER_URLS", "")
	path := filepath.Join(t.TempDir(), "models.json")
	assert.NoError(t, os.WriteFile(path, []byte(`{
		"openai/model":{"endpoint":"unsupported"}
	}`), 0o600))
	t.Setenv("MODEL_CAPABILITIES", path)
	_, _, err = openai.LoadConfig()
	assert.ErrorIs(t, err, openai.ErrInvalidModelEndpoint)
}
