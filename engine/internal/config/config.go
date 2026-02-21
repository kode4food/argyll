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

type (
	// Config holds configuration settings for the orchestrator
	Config struct {
		// API Server
		APIHost        string
		APIPort        int
		WebhookBaseURL string
		LogLevel       string

		// Stores & Archiving
		CatalogStore   timebox.StoreConfig
		PartitionStore timebox.StoreConfig
		FlowStore      timebox.StoreConfig

		// Work & Retry
		Work api.WorkConfig

		// Engine
		StepTimeout     int64
		FlowCacheSize   int
		MemoCacheSize   int
		ShutdownTimeout time.Duration
	}
)

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
	DefaultMemoCacheSize       = 10240

	DefaultRetryMaxRetries  = 10
	DefaultRetryBackoff     = 1000
	DefaultMaxRetryBackoff  = 60000
	DefaultRetryBackoffType = api.BackoffTypeExponential
)

var (
	ErrInvalidAPIPort         = errors.New("invalid API port")
	ErrInvalidStepTimeout     = errors.New("step timeout must be positive")
	ErrInvalidRetryMaxRetries = errors.New(
		"retry max retries cannot be zero",
	)
	ErrInvalidRetryBackoff = errors.New(
		"retry backoff must be positive",
	)
	ErrInvalidRetryMaxBackoff = errors.New(
		"retry max backoff must be positive",
	)
	ErrRetryBackoffTooSmall = errors.New(
		"retry max backoff must be >= retry backoff",
	)
	ErrInvalidRetryBackoffType = errors.New("invalid retry backoff type")
)

// NewDefaultConfig creates a configuration with sensible defaults for all
// engine settings, stores, and retry behavior
func NewDefaultConfig() *Config {
	return &Config{
		APIPort:        DefaultAPIPort,
		APIHost:        DefaultAPIHost,
		WebhookBaseURL: "http://localhost:8080",
		CatalogStore: timebox.StoreConfig{
			Addr:         DefaultRedisEndpoint,
			Password:     "",
			DB:           DefaultRedisDB,
			Prefix:       DefaultRedisPrefix,
			WorkerCount:  DefaultSnapshotWorkers,
			MaxQueueSize: DefaultSnapshotQueueSize,
			SaveTimeout:  DefaultSnapshotSaveTimeout,
			TrimEvents:   true,
		},
		PartitionStore: timebox.StoreConfig{
			Addr:         DefaultRedisEndpoint,
			Password:     "",
			DB:           DefaultRedisDB,
			Prefix:       DefaultRedisPrefix,
			WorkerCount:  DefaultSnapshotWorkers,
			MaxQueueSize: DefaultSnapshotQueueSize,
			SaveTimeout:  DefaultSnapshotSaveTimeout,
			TrimEvents:   true,
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
		Work: api.WorkConfig{
			MaxRetries:  DefaultRetryMaxRetries,
			Backoff:     DefaultRetryBackoff,
			MaxBackoff:  DefaultMaxRetryBackoff,
			BackoffType: DefaultRetryBackoffType,
		},
		StepTimeout:     DefaultStepTimeout,
		FlowCacheSize:   DefaultCacheSize,
		MemoCacheSize:   DefaultMemoCacheSize,
		ShutdownTimeout: DefaultShutdownTimeout,
		LogLevel:        "info",
	}
}

// LoadFromEnv populates configuration values from environment variables
func (c *Config) LoadFromEnv() {
	LoadStoreConfigFromEnv(&c.CatalogStore, "CATALOG")
	LoadStoreConfigFromEnv(&c.PartitionStore, "PARTITION")
	LoadStoreConfigFromEnv(&c.FlowStore, "PARTITION")

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
	if flowSizeStr := os.Getenv("FLOW_CACHE_SIZE"); flowSizeStr != "" {
		cacheSize, err := strconv.Atoi(flowSizeStr)
		if err == nil && cacheSize > 0 {
			c.FlowCacheSize = cacheSize
		}
	}
	if memoSizeStr := os.Getenv("MEMO_CACHE_SIZE"); memoSizeStr != "" {
		cacheSize, err := strconv.Atoi(memoSizeStr)
		if err == nil && cacheSize > 0 {
			c.MemoCacheSize = cacheSize
		}
	}
	if timeout := os.Getenv("STEP_TIMEOUT"); timeout != "" {
		if ms, err := strconv.ParseInt(timeout, 10, 64); err == nil && ms > 0 {
			c.StepTimeout = ms
		}
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.LogLevel = logLevel
	}

	if maxRetries := os.Getenv("RETRY_MAX_RETRIES"); maxRetries != "" {
		if retries, err := strconv.Atoi(maxRetries); err == nil {
			c.Work.MaxRetries = retries
		}
	}
	if backoff := os.Getenv("RETRY_BACKOFF"); backoff != "" {
		if ms, err := strconv.ParseInt(backoff, 10, 64); err == nil {
			c.Work.Backoff = ms
		}
	}
	if maxBackoff := os.Getenv("RETRY_MAX_BACKOFF"); maxBackoff != "" {
		if ms, err := strconv.ParseInt(maxBackoff, 10, 64); err == nil {
			c.Work.MaxBackoff = ms
		}
	}
	if backoffType := os.Getenv("RETRY_BACKOFF_TYPE"); backoffType != "" {
		c.Work.BackoffType = backoffType
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

	if c.Work.MaxRetries == 0 {
		return ErrInvalidRetryMaxRetries
	}

	if c.Work.Backoff <= 0 {
		return ErrInvalidRetryBackoff
	}

	if c.Work.MaxBackoff <= 0 {
		return ErrInvalidRetryMaxBackoff
	}

	if c.Work.MaxBackoff < c.Work.Backoff {
		return ErrRetryBackoffTooSmall
	}

	if c.Work.BackoffType != api.BackoffTypeFixed &&
		c.Work.BackoffType != api.BackoffTypeLinear &&
		c.Work.BackoffType != api.BackoffTypeExponential {
		return fmt.Errorf("%w: %s",
			ErrInvalidRetryBackoffType, c.Work.BackoffType)
	}

	return nil
}

// LoadStoreConfigFromEnv loads Redis store configuration from environment
// variables with the given prefix (e.g., "CATALOG" or "PARTITION")
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
