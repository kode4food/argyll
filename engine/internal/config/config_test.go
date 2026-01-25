package config_test

import (
	"os"
	"testing"

	"github.com/kode4food/timebox"
	testify "github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert"
	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/config"
)

func TestConfigValidation(t *testing.T) {
	as := assert.New(t)

	t.Run("valid_default_config", func(t *testing.T) {
		cfg := config.NewDefaultConfig()
		as.ConfigValid(cfg)
	})

	t.Run("valid_test_config", func(t *testing.T) {
		cfg := helpers.NewTestConfig()
		as.ConfigValid(cfg)
	})

	tests := []struct {
		name          string
		configMod     func(*config.Config)
		errorContains string
	}{
		{
			name: "invalid_api_port_zero",
			configMod: func(c *config.Config) {
				c.APIPort = 0
			},
			errorContains: "invalid API port",
		},
		{
			name: "invalid_api_port_negative",
			configMod: func(c *config.Config) {
				c.APIPort = -1
			},
			errorContains: "invalid API port",
		},
		{
			name: "invalid_api_port_too_high",
			configMod: func(c *config.Config) {
				c.APIPort = 70000
			},
			errorContains: "invalid API port",
		},
		{
			name: "zero_step_timeout",
			configMod: func(c *config.Config) {
				c.StepTimeout = 0
			},
			errorContains: "step timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := helpers.NewTestConfig()
			tt.configMod(cfg)
			as.ConfigInvalid(cfg, tt.errorContains)
		})
	}
}

func TestDefaultConfigValues(t *testing.T) {
	as := assert.New(t)

	cfg := config.NewDefaultConfig()

	as.Equal(config.DefaultAPIPort, cfg.APIPort)
	as.Equal("0.0.0.0", cfg.APIHost)
	as.Equal(config.DefaultStepTimeout, cfg.StepTimeout)
	as.Equal(config.DefaultShutdownTimeout, cfg.ShutdownTimeout)
	as.Equal("info", cfg.LogLevel)
}

func TestStoreLoadFromEnv(t *testing.T) {
	tests := []struct {
		envVars          map[string]string
		name             string
		envPrefix        string
		checkAddr        string
		checkPassword    string
		checkPrefix      string
		checkDB          int
		checkWorkerCount *int
	}{
		{
			name:      "load_all_fields",
			envPrefix: "TEST",
			envVars: map[string]string{
				"TEST_REDIS_ADDR":       "redis.example.com:6379",
				"TEST_REDIS_PASSWORD":   "secret123",
				"TEST_REDIS_DB":         "5",
				"TEST_REDIS_PREFIX":     "custom-prefix",
				"TEST_SNAPSHOT_WORKERS": "6",
			},
			checkAddr:        "redis.example.com:6379",
			checkPassword:    "secret123",
			checkDB:          5,
			checkPrefix:      "custom-prefix",
			checkWorkerCount: func() *int { v := 6; return &v }(),
		},
		{
			name:      "load_addr_only",
			envPrefix: "APP",
			envVars: map[string]string{
				"APP_REDIS_ADDR": "localhost:9999",
			},
			checkAddr:     "localhost:9999",
			checkPassword: "",
			checkDB:       0,
			checkPrefix:   "",
		},
		{
			name:      "load_worker_zero",
			envPrefix: "ZERO",
			envVars: map[string]string{
				"ZERO_SNAPSHOT_WORKERS": "0",
			},
			checkWorkerCount: func() *int { v := 0; return &v }(),
		},
		{
			name:      "load_with_invalid_db",
			envPrefix: "INVALID",
			envVars: map[string]string{
				"INVALID_REDIS_DB": "not_a_number",
			},
			checkDB: 0,
		},
		{
			name:      "invalid_worker_ignored",
			envPrefix: "BADWORKER",
			envVars: map[string]string{
				"BADWORKER_SNAPSHOT_WORKERS": "not_a_number",
			},
		},
		{
			name:      "no_env_vars",
			envPrefix: "NONE",
			envVars:   map[string]string{},
			checkAddr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as := assert.New(t)

			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
				t.Cleanup(func() { _ = os.Unsetenv(key) })
			}

			storeConfig := &timebox.StoreConfig{}
			config.LoadStoreConfigFromEnv(storeConfig, tt.envPrefix)

			if tt.checkAddr != "" {
				as.Equal(tt.checkAddr, storeConfig.Addr)
			}
			if tt.checkPassword != "" {
				as.Equal(tt.checkPassword, storeConfig.Password)
			}
			if tt.envVars[tt.envPrefix+"_REDIS_DB"] != "" {
				as.Equal(tt.checkDB, storeConfig.DB)
			}
			if tt.checkPrefix != "" {
				as.Equal(tt.checkPrefix, storeConfig.Prefix)
			}
			if tt.checkWorkerCount != nil {
				as.Equal(*tt.checkWorkerCount, storeConfig.WorkerCount)
			}
		})
	}
}

func TestValidateValidEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*config.Config)
	}{
		{
			name:   "min_valid_port",
			modify: func(c *config.Config) { c.APIPort = 1 },
		},
		{
			name:   "max_valid_port",
			modify: func(c *config.Config) { c.APIPort = 65535 },
		},
		{
			name:   "one_nanosecond_timeout",
			modify: func(c *config.Config) { c.StepTimeout = 1 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewDefaultConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			testify.NoError(t, err)
		})
	}
}

func TestValidateNegativeTimeout(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.StepTimeout = -1

	err := cfg.Validate()
	testify.Error(t, err)
	testify.ErrorIs(t, err, config.ErrInvalidStepTimeout)
}

func TestConfigLoadFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		check   func(*testing.T, *config.Config)
	}{
		{
			name: "load_api_port",
			envVars: map[string]string{
				"API_PORT": "9090",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t, 9090, c.APIPort)
			},
		},
		{
			name: "load_api_host",
			envVars: map[string]string{
				"API_HOST": "127.0.0.1",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t, "127.0.0.1", c.APIHost)
			},
		},
		{
			name: "load_webhook_base_url",
			envVars: map[string]string{
				"WEBHOOK_BASE_URL": "http://webhooks.example.com",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t,
					"http://webhooks.example.com", c.WebhookBaseURL,
				)
			},
		},
		{
			name: "load_flow_cache_size",
			envVars: map[string]string{
				"FLOW_CACHE_SIZE": "8192",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t, 8192, c.FlowCacheSize)
			},
		},
		{
			name: "load_log_level",
			envVars: map[string]string{
				"LOG_LEVEL": "debug",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t, "debug", c.LogLevel)
			},
		},
		{
			name: "invalid_api_port_ignored",
			envVars: map[string]string{
				"API_PORT": "not_a_number",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t, config.DefaultAPIPort, c.APIPort)
			},
		},
		{
			name: "invalid_cache_size_ignored",
			envVars: map[string]string{
				"FLOW_CACHE_SIZE": "invalid",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t, config.DefaultCacheSize, c.FlowCacheSize)
			},
		},
		{
			name: "zero_cache_size_ignored",
			envVars: map[string]string{
				"FLOW_CACHE_SIZE": "0",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t, config.DefaultCacheSize, c.FlowCacheSize)
			},
		},
		{
			name: "load_retry_max_retries",
			envVars: map[string]string{
				"RETRY_MAX_RETRIES": "5",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t, 5, c.Work.MaxRetries)
			},
		},
		{
			name: "load_retry_backoff_ms",
			envVars: map[string]string{
				"RETRY_BACKOFF_MS": "2000",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t, int64(2000), c.Work.BackoffMs)
			},
		},
		{
			name: "load_retry_max_backoff_ms",
			envVars: map[string]string{
				"RETRY_MAX_BACKOFF_MS": "60000",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t, int64(60000), c.Work.MaxBackoffMs)
			},
		},
		{
			name: "load_retry_backoff_type",
			envVars: map[string]string{
				"RETRY_BACKOFF_TYPE": "exponential",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t, "exponential", c.Work.BackoffType)
			},
		},
		{
			name: "invalid_retry_max_retries_ignored",
			envVars: map[string]string{
				"RETRY_MAX_RETRIES": "not_a_number",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t,
					config.DefaultRetryMaxRetries, c.Work.MaxRetries,
				)
			},
		},
		{
			name: "invalid_retry_backoff_ms_ignored",
			envVars: map[string]string{
				"RETRY_BACKOFF_MS": "invalid",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t,
					int64(config.DefaultRetryBackoffMs), c.Work.BackoffMs,
				)
			},
		},
		{
			name: "invalid_retry_max_backoff_ms_ignored",
			envVars: map[string]string{
				"RETRY_MAX_BACKOFF_MS": "bad_value",
			},
			check: func(t *testing.T, c *config.Config) {
				testify.Equal(t,
					int64(config.DefaultRetryMaxBackoffMs), c.Work.MaxBackoffMs,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
				t.Cleanup(func() { _ = os.Unsetenv(key) })
			}

			cfg := config.NewDefaultConfig()
			cfg.LoadFromEnv()
			tt.check(t, cfg)
		})
	}
}
