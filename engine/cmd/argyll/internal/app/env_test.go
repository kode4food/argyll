package app_test

import (
	"testing"

	"github.com/kode4food/timebox/raft"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/cmd/argyll/internal/app"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/config"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("RAFT_NODE_ID", "node-2")
	t.Setenv("API_PORT", "9090")
	t.Setenv("STEP_TIMEOUT", "45000")

	cfg, err := app.LoadConfig()
	assert.NoError(t, err)
	assert.Equal(t, "node-2", cfg.Raft.LocalID)
	assert.Equal(t, 9090, cfg.Server.APIPort)
	assert.Equal(t, api.NodeID("node-2"), cfg.Engine.NodeID)
	assert.Equal(t, []api.NodeID{"node-2"}, cfg.Engine.Nodes)
	assert.Equal(t, int64(45000), cfg.Engine.StepTimeout)
}

func TestLoadServerConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr error
		check   func(*testing.T, app.ServerConfig)
	}{
		{
			name: "defaults",
			check: func(t *testing.T, c app.ServerConfig) {
				assert.Equal(t, app.DefaultAPIHost, c.APIHost)
				assert.Equal(t, app.DefaultAPIPort, c.APIPort)
				assert.Equal(t, app.DefaultLogLevel, c.LogLevel)
				assert.Equal(t,
					app.DefaultShutdownTimeout, c.ShutdownTimeout,
				)
			},
		},
		{
			name: "load_api_port",
			envVars: map[string]string{
				"API_PORT": "9090",
			},
			check: func(t *testing.T, c app.ServerConfig) {
				assert.Equal(t, 9090, c.APIPort)
			},
		},
		{
			name: "load_api_host",
			envVars: map[string]string{
				"API_HOST": "127.0.0.1",
			},
			check: func(t *testing.T, c app.ServerConfig) {
				assert.Equal(t, "127.0.0.1", c.APIHost)
			},
		},
		{
			name: "load_log_level",
			envVars: map[string]string{
				"LOG_LEVEL": "debug",
			},
			check: func(t *testing.T, c app.ServerConfig) {
				assert.Equal(t, "debug", c.LogLevel)
			},
		},
		{
			name: "invalid_api_port_errors",
			envVars: map[string]string{
				"API_PORT": "not_a_number",
			},
			wantErr: app.ErrInvalidEnvValue,
		},
		{
			name: "api_port_too_high_errors",
			envVars: map[string]string{
				"API_PORT": "70000",
			},
			wantErr: app.ErrEnvOutOfRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.envVars)

			cfg, err := app.LoadServerConfig()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			tt.check(t, cfg)
		})
	}
}

func TestLoadEngineConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr error
		check   func(*testing.T, *config.Config)
	}{
		{
			name: "raft_identity",
			check: func(t *testing.T, c *config.Config) {
				assert.Equal(t, api.NodeID("node-2"), c.NodeID)
				assert.Equal(t,
					[]api.NodeID{"node-1", "node-2"}, c.Nodes,
				)
			},
		},
		{
			name: "load_webhook_base_url",
			envVars: map[string]string{
				"WEBHOOK_BASE_URL": "http://webhooks.example.com",
			},
			check: func(t *testing.T, c *config.Config) {
				assert.Equal(t,
					"http://webhooks.example.com", c.WebhookBaseURL,
				)
			},
		},
		{
			name: "load_timebox_cache_size",
			envVars: map[string]string{
				"TIMEBOX_CACHE_SIZE": "8192",
			},
			check: func(t *testing.T, c *config.Config) {
				assert.Equal(t, 8192, c.Timebox.CacheSize)
				assert.Equal(t, 8192, c.FlowStoreConfig().CacheSize)
				assert.Equal(t, config.EngineStoreCacheSize,
					c.EngineStoreConfig().CacheSize)
			},
		},
		{
			name: "load_step_timeout",
			envVars: map[string]string{
				"STEP_TIMEOUT": "45000",
			},
			check: func(t *testing.T, c *config.Config) {
				assert.Equal(t, int64(45000), c.StepTimeout)
			},
		},
		{
			name: "load_retry_max_retries",
			envVars: map[string]string{
				"RETRY_MAX_RETRIES": "5",
			},
			check: func(t *testing.T, c *config.Config) {
				assert.Equal(t, 5, c.Work.MaxRetries)
			},
		},
		{
			name: "load_retry_backoff",
			envVars: map[string]string{
				"RETRY_INITIAL_BACKOFF": "2000",
			},
			check: func(t *testing.T, c *config.Config) {
				assert.Equal(t, int64(2000), c.Work.InitBackoff)
			},
		},
		{
			name: "load_retry_max_backoff",
			envVars: map[string]string{
				"RETRY_MAX_BACKOFF": "60000",
			},
			check: func(t *testing.T, c *config.Config) {
				assert.Equal(t, int64(60000), c.Work.MaxBackoff)
			},
		},
		{
			name: "load_retry_backoff_type",
			envVars: map[string]string{
				"RETRY_BACKOFF_TYPE": "exponential",
			},
			check: func(t *testing.T, c *config.Config) {
				assert.Equal(t, "exponential", c.Work.BackoffType)
			},
		},
		{
			name: "invalid_cache_size_errors",
			envVars: map[string]string{
				"TIMEBOX_CACHE_SIZE": "invalid",
			},
			wantErr: app.ErrInvalidEnvValue,
		},
		{
			name: "zero_cache_size_errors",
			envVars: map[string]string{
				"TIMEBOX_CACHE_SIZE": "0",
			},
			wantErr: app.ErrEnvOutOfRange,
		},
		{
			name: "invalid_step_timeout_errors",
			envVars: map[string]string{
				"STEP_TIMEOUT": "invalid",
			},
			wantErr: app.ErrInvalidEnvValue,
		},
		{
			name: "non_positive_step_timeout_errors",
			envVars: map[string]string{
				"STEP_TIMEOUT": "0",
			},
			wantErr: app.ErrEnvOutOfRange,
		},
		{
			name: "invalid_retry_max_retries_errors",
			envVars: map[string]string{
				"RETRY_MAX_RETRIES": "not_a_number",
			},
			wantErr: app.ErrInvalidEnvValue,
		},
		{
			name: "invalid_retry_backoff_errors",
			envVars: map[string]string{
				"RETRY_INITIAL_BACKOFF": "invalid",
			},
			wantErr: app.ErrInvalidEnvValue,
		},
		{
			name: "invalid_retry_max_backoff_errors",
			envVars: map[string]string{
				"RETRY_MAX_BACKOFF": "bad_value",
			},
			wantErr: app.ErrInvalidEnvValue,
		},
	}

	rc := raft.Config{
		LocalID: "node-2",
		Servers: []raft.Server{{ID: "node-1"}, {ID: "node-2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.envVars)

			cfg, err := app.LoadEngineConfig(rc)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			tt.check(t, cfg)
		})
	}
}

func setEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	for key, value := range vars {
		t.Setenv(key, value)
	}
}
