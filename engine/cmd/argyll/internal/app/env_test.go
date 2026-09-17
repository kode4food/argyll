package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/cmd/argyll/internal/app"
	"github.com/kode4food/argyll/engine/pkg/config"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr error
		check   func(*testing.T, app.Config)
	}{
		{
			name: "defaults",
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, app.DefaultAPIHost, c.APIHost)
				assert.Equal(t, app.DefaultAPIPort, c.APIPort)
				assert.Equal(t,
					app.DefaultWebhookBaseURL, c.WebhookBaseURL,
				)
				assert.Equal(t, app.DefaultLogLevel, c.LogLevel)
				assert.Equal(t,
					app.DefaultShutdownTimeout, c.ShutdownTimeout,
				)
				assert.Equal(t, config.DefaultStepTimeout, c.Engine.StepTimeout)
			},
		},
		{
			name: "load_api_port",
			envVars: map[string]string{
				"API_PORT": "9090",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, 9090, c.APIPort)
			},
		},
		{
			name: "load_api_host",
			envVars: map[string]string{
				"API_HOST": "127.0.0.1",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, "127.0.0.1", c.APIHost)
			},
		},
		{
			name: "load_webhook_base_url",
			envVars: map[string]string{
				"WEBHOOK_BASE_URL": "http://webhooks.example.com",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t,
					"http://webhooks.example.com", c.WebhookBaseURL,
				)
			},
		},
		{
			name: "load_log_level",
			envVars: map[string]string{
				"LOG_LEVEL": "debug",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, "debug", c.LogLevel)
			},
		},
		{
			name: "load_timebox_cache_size",
			envVars: map[string]string{
				"TIMEBOX_CACHE_SIZE": "8192",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, 8192, c.Engine.Timebox.CacheSize)
				assert.Equal(t,
					8192, c.Engine.FlowStoreConfig().CacheSize,
				)
				assert.Equal(t, config.EngineStoreCacheSize,
					c.Engine.EngineStoreConfig().CacheSize)
			},
		},
		{
			name: "load_step_timeout",
			envVars: map[string]string{
				"STEP_TIMEOUT": "45000",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, int64(45000), c.Engine.StepTimeout)
			},
		},
		{
			name: "load_retry_max_retries",
			envVars: map[string]string{
				"RETRY_MAX_RETRIES": "5",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, 5, c.Engine.Work.MaxRetries)
			},
		},
		{
			name: "load_retry_backoff",
			envVars: map[string]string{
				"RETRY_INITIAL_BACKOFF": "2000",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, int64(2000), c.Engine.Work.InitBackoff)
			},
		},
		{
			name: "load_retry_max_backoff",
			envVars: map[string]string{
				"RETRY_MAX_BACKOFF": "60000",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, int64(60000), c.Engine.Work.MaxBackoff)
			},
		},
		{
			name: "load_retry_backoff_type",
			envVars: map[string]string{
				"RETRY_BACKOFF_TYPE": "exponential",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, "exponential", c.Engine.Work.BackoffType)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.envVars)

			cfg, err := app.LoadConfig()
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
