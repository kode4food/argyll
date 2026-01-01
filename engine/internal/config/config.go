package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// Config holds configuration settings for the orchestrator
type Config struct {
	APIHost        string
	WebhookBaseURL string
	LogLevel       string
	EngineStore    timebox.StoreConfig
	FlowStore      timebox.StoreConfig
	WorkConfig     api.WorkConfig
	APIPort        int
	StepTimeout    int64

	FlowCacheSize      int
	ShutdownTimeout    time.Duration
	RetryCheckInterval time.Duration

	HibernatorURL    string
	HibernatorPrefix string
}

const (
	DefaultStepTimeout     = 30 * api.Second
	DefaultShutdownTimeout = 10 * time.Second

	DefaultAPIPort = 8080
	DefaultAPIHost = "0.0.0.0"
	MaxTCPPort     = 65535
	DefaultRedisDB = 0

	DefaultRedisEndpoint       = "localhost:6379"
	DefaultRedisPrefix         = "argyll"
	DefaultSnapshotWorkers     = 4
	DefaultSnapshotQueueSize   = 1000
	DefaultSnapshotSaveTimeout = 30 * time.Second
	DefaultCacheSize           = 4096

	DefaultRetryCheckInterval = 1 * time.Second
	DefaultRetryMaxRetries    = 10
	DefaultRetryBackoffMs     = 1000
	DefaultRetryMaxBackoffMs  = 60000
	DefaultRetryBackoffType   = api.BackoffTypeExponential
)

var (
	ErrInvalidAPIPort     = errors.New("invalid API port")
	ErrInvalidStepTimeout = errors.New("step timeout must be positive")
)

// NewDefaultConfig creates a configuration with sensible defaults for all
// engine settings, stores, and retry behavior
func NewDefaultConfig() *Config {
	return &Config{
		APIPort:        DefaultAPIPort,
		APIHost:        DefaultAPIHost,
		WebhookBaseURL: "http://localhost:8080",
		EngineStore: timebox.StoreConfig{
			Addr:         DefaultRedisEndpoint,
			Password:     "",
			DB:           DefaultRedisDB,
			Prefix:       DefaultRedisPrefix,
			WorkerCount:  DefaultSnapshotWorkers,
			MaxQueueSize: DefaultSnapshotQueueSize,
			SaveTimeout:  DefaultSnapshotSaveTimeout,
		},
		FlowStore: timebox.StoreConfig{
			Addr:         DefaultRedisEndpoint,
			Password:     "",
			DB:           DefaultRedisDB,
			Prefix:       DefaultRedisPrefix,
			WorkerCount:  DefaultSnapshotWorkers,
			MaxQueueSize: DefaultSnapshotQueueSize,
			SaveTimeout:  DefaultSnapshotSaveTimeout,
		},
		WorkConfig: api.WorkConfig{
			MaxRetries:   DefaultRetryMaxRetries,
			BackoffMs:    DefaultRetryBackoffMs,
			MaxBackoffMs: DefaultRetryMaxBackoffMs,
			BackoffType:  DefaultRetryBackoffType,
		},
		StepTimeout:        DefaultStepTimeout,
		FlowCacheSize:      DefaultCacheSize,
		ShutdownTimeout:    DefaultShutdownTimeout,
		RetryCheckInterval: DefaultRetryCheckInterval,
		LogLevel:           "info",
	}
}

// LoadFromEnv populates configuration values from environment variables
func (c *Config) LoadFromEnv() {
	LoadStoreConfigFromEnv(&c.EngineStore, "ENGINE")
	LoadStoreConfigFromEnv(&c.FlowStore, "FLOW")

	if apiPort := os.Getenv("API_PORT"); apiPort != "" {
		if port, err := strconv.Atoi(apiPort); err == nil {
			c.APIPort = port
		}
	}
	if apiHost := os.Getenv("API_HOST"); apiHost != "" {
		c.APIHost = apiHost
	}
	if webhookBaseURL := os.Getenv("WEBHOOK_BASE_URL"); webhookBaseURL != "" {
		c.WebhookBaseURL = webhookBaseURL
	}
	if cacheSizeStr := os.Getenv("FLOW_CACHE_SIZE"); cacheSizeStr != "" {
		cacheSize, err := strconv.Atoi(cacheSizeStr)
		if err == nil && cacheSize > 0 {
			c.FlowCacheSize = cacheSize
		}
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.LogLevel = logLevel
	}

	if maxRetries := os.Getenv("RETRY_MAX_RETRIES"); maxRetries != "" {
		if retries, err := strconv.Atoi(maxRetries); err == nil {
			c.WorkConfig.MaxRetries = retries
		}
	}
	if backoffMs := os.Getenv("RETRY_BACKOFF_MS"); backoffMs != "" {
		if ms, err := strconv.ParseInt(backoffMs, 10, 64); err == nil {
			c.WorkConfig.BackoffMs = ms
		}
	}
	if maxBackoffMs := os.Getenv("RETRY_MAX_BACKOFF_MS"); maxBackoffMs != "" {
		if ms, err := strconv.ParseInt(maxBackoffMs, 10, 64); err == nil {
			c.WorkConfig.MaxBackoffMs = ms
		}
	}
	if backoffType := os.Getenv("RETRY_BACKOFF_TYPE"); backoffType != "" {
		c.WorkConfig.BackoffType = backoffType
	}

	if hibernatorURL := os.Getenv("HIBERNATOR_URL"); hibernatorURL != "" {
		c.HibernatorURL = hibernatorURL
	}
	if hibernatorPfx := os.Getenv("HIBERNATOR_PREFIX"); hibernatorPfx != "" {
		c.HibernatorPrefix = hibernatorPfx
	}
}

// Validate checks that all configuration values are valid
func (c *Config) Validate() error {
	if c.APIPort <= 0 || c.APIPort > MaxTCPPort {
		return fmt.Errorf("%w: %d", ErrInvalidAPIPort, c.APIPort)
	}

	if c.StepTimeout <= 0 {
		return ErrInvalidStepTimeout
	}

	return nil
}

// LoadStoreConfigFromEnv loads Redis store configuration from environment
// variables with the given prefix (e.g., "ENGINE" or "FLOW")
func LoadStoreConfigFromEnv(s *timebox.StoreConfig, prefix string) {
	if addr := os.Getenv(prefix + "_REDIS_ADDR"); addr != "" {
		s.Addr = addr
	}
	if password := os.Getenv(prefix + "_REDIS_PASSWORD"); password != "" {
		s.Password = password
	}
	if dbStr := os.Getenv(prefix + "_REDIS_DB"); dbStr != "" {
		db, err := strconv.Atoi(dbStr)
		if err == nil {
			s.DB = db
		}
	}
	if envPrefix := os.Getenv(prefix + "_REDIS_PREFIX"); envPrefix != "" {
		s.Prefix = envPrefix
	}
	if envCount := os.Getenv(prefix + "_SNAPSHOT_WORKERS"); envCount != "" {
		if wc, err := strconv.Atoi(envCount); err == nil && wc >= 0 {
			s.WorkerCount = wc
		}
	}
}
