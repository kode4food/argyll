package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

// Config holds configuration settings for the workflow engine
type Config struct {
	APIHost            string
	WebhookBaseURL     string
	LogLevel           string
	EngineStore        timebox.StoreConfig
	WorkflowStore      timebox.StoreConfig
	WorkConfig         api.WorkConfig
	APIPort            int
	StepTimeout        int64
	MaxWorkflows       int
	MaxStateKeySize    int
	MaxStateValueSize  int
	WorkflowCacheSize  int
	ShutdownTimeout    time.Duration
	RetryCheckInterval time.Duration
}

const (
	DefaultStepTimeout     = 30 * api.Second
	DefaultShutdownTimeout = 10 * time.Second

	DefaultMaxWorkflows      = 1000
	DefaultMaxStateKeySize   = 1024 * 1024
	DefaultMaxStateValueSize = 10 * 1024 * 1024

	DefaultAPIPort = 8080
	DefaultAPIHost = "0.0.0.0"
	MaxTCPPort     = 65535
	DefaultRedisDB = 0

	DefaultRedisEndpoint       = "localhost:6379"
	DefaultRedisPrefix         = "spuds"
	DefaultSnapshotWorkers     = 4
	DefaultSnapshotQueueSize   = 1000
	DefaultSnapshotSaveTimeout = 30 * time.Second
	DefaultCacheSize           = 4096
	DefaultRetryCheckInterval  = 1 * time.Second

	DefaultRetryMaxRetries   = 3
	DefaultRetryBackoffMs    = 1000
	DefaultRetryMaxBackoffMs = 60000
	DefaultRetryBackoffType  = api.BackoffTypeExponential
)

var (
	ErrInvalidAPIPort      = errors.New("invalid API port")
	ErrInvalidStepTimeout  = errors.New("step timeout must be positive")
	ErrInvalidMaxWorkflows = errors.New("max workflows must be positive")
	ErrInvalidMaxKey       = errors.New("max state key size must be positive")
	ErrInvalidMaxValue     = errors.New("max state value size must be positive")
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
		WorkflowStore: timebox.StoreConfig{
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
		MaxWorkflows:       DefaultMaxWorkflows,
		MaxStateKeySize:    DefaultMaxStateKeySize,
		MaxStateValueSize:  DefaultMaxStateValueSize,
		WorkflowCacheSize:  DefaultCacheSize,
		ShutdownTimeout:    DefaultShutdownTimeout,
		RetryCheckInterval: DefaultRetryCheckInterval,
		LogLevel:           "info",
	}
}

func (c *Config) LoadFromEnv() {
	LoadStoreConfigFromEnv(&c.EngineStore, "ENGINE")
	LoadStoreConfigFromEnv(&c.WorkflowStore, "WORKFLOW")

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
	if cacheSizeStr := os.Getenv("WORKFLOW_CACHE_SIZE"); cacheSizeStr != "" {
		cacheSize, err := strconv.Atoi(cacheSizeStr)
		if err == nil && cacheSize > 0 {
			c.WorkflowCacheSize = cacheSize
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
}

func (c *Config) Validate() error {
	if c.APIPort <= 0 || c.APIPort > MaxTCPPort {
		return fmt.Errorf("%w: %d", ErrInvalidAPIPort, c.APIPort)
	}

	if c.StepTimeout <= 0 {
		return ErrInvalidStepTimeout
	}

	if c.MaxWorkflows <= 0 {
		return ErrInvalidMaxWorkflows
	}

	if c.MaxStateKeySize <= 0 {
		return ErrInvalidMaxKey
	}

	if c.MaxStateValueSize <= 0 {
		return ErrInvalidMaxValue
	}

	return nil
}

// LoadStoreConfigFromEnv loads Redis store configuration from environment
// variables with the given prefix (e.g., "ENGINE" or "WORKFLOW")
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
}
