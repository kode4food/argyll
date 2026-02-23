package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
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
	DefaultRetryInitBackoff = 1000
	DefaultMaxRetryBackoff  = 60000
	DefaultRetryBackoffType = api.BackoffTypeExponential

	MaxFlowCacheSize    = 1_000_000
	MaxMemoCacheSize    = 10_000_000
	MaxRetryMaxRetries  = 1000
	MaxStepTimeout      = 365 * 24 * 60 * api.Minute // 1 year in ms
	MaxRetryInitBackoff = 24 * 60 * api.Minute       // 1 day in ms
	MaxRetryMaxBackoff  = MaxRetryInitBackoff
)

var (
	ErrInvalidAPIPort         = errors.New("invalid API port")
	ErrInvalidStepTimeout     = errors.New("step timeout must be positive")
	ErrInvalidRetryMaxRetries = errors.New(
		"retry max retries cannot be zero",
	)
	ErrInvalidRetryInitBackoff = errors.New(
		"retry initial backoff must be positive",
	)
	ErrInvalidRetryMaxBackoff = errors.New(
		"retry max backoff must be positive",
	)
	ErrRetryMaxBackoffTooSmall = errors.New(
		"retry max backoff must be >= retry initial backoff",
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
			JoinKey:      events.FlowJoinKey,
			ParseKey:     events.FlowParseKey,
		},
		Work: api.WorkConfig{
			MaxRetries:  DefaultRetryMaxRetries,
			InitBackoff: DefaultRetryInitBackoff,
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

// LoadFromEnv populates configuration values from environment variables.
// Returns an error if any env var cannot be parsed.
func (c *Config) LoadFromEnv() error {
	LoadStoreConfigFromEnv(&c.CatalogStore, "CATALOG")
	LoadStoreConfigFromEnv(&c.PartitionStore, "PARTITION")
	LoadStoreConfigFromEnv(&c.FlowStore, "PARTITION")

	if apiHost := os.Getenv("API_HOST"); apiHost != "" {
		c.APIHost = apiHost
	}
	if webhookBaseURL := os.Getenv("WEBHOOK_BASE_URL"); webhookBaseURL != "" {
		c.WebhookBaseURL = webhookBaseURL
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.LogLevel = logLevel
	}
	if backoffType := os.Getenv("RETRY_BACKOFF_TYPE"); backoffType != "" {
		c.Work.BackoffType = backoffType
	}

	if err := loadEnvInt("API_PORT", &c.APIPort, 0, MaxTCPPort); err != nil {
		return err
	}

	if err := loadEnvInt(
		"FLOW_CACHE_SIZE", &c.FlowCacheSize, 0, MaxFlowCacheSize,
	); err != nil {
		return err
	}
	if err := loadEnvInt(
		"MEMO_CACHE_SIZE", &c.MemoCacheSize, 0, MaxMemoCacheSize,
	); err != nil {
		return err
	}
	if err := loadEnvInt(
		"STEP_TIMEOUT", &c.StepTimeout, 0, MaxStepTimeout,
	); err != nil {
		return err
	}

	if err := loadEnvInt(
		"RETRY_MAX_RETRIES", &c.Work.MaxRetries, 0, MaxRetryMaxRetries,
	); err != nil {
		return err
	}
	if err := loadEnvInt(
		"RETRY_INITIAL_BACKOFF", &c.Work.InitBackoff, 0, MaxRetryInitBackoff,
	); err != nil {
		return err
	}
	if err := loadEnvInt(
		"RETRY_MAX_BACKOFF", &c.Work.MaxBackoff, 0, MaxRetryMaxBackoff,
	); err != nil {
		return err
	}

	return nil
}

// WithWorkDefaults returns a copy of the config with zero-valued work fields
// filled in from defaults
func (c *Config) WithWorkDefaults() *Config {
	res := *c
	if res.Work.MaxRetries == 0 {
		res.Work.MaxRetries = DefaultRetryMaxRetries
	}
	if res.Work.InitBackoff <= 0 {
		res.Work.InitBackoff = DefaultRetryInitBackoff
	}
	if res.Work.MaxBackoff <= 0 {
		res.Work.MaxBackoff = DefaultMaxRetryBackoff
	}
	if res.Work.BackoffType == "" {
		res.Work.BackoffType = DefaultRetryBackoffType
	}
	return &res
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

	if c.Work.InitBackoff <= 0 {
		return ErrInvalidRetryInitBackoff
	}

	if c.Work.MaxBackoff <= 0 {
		return ErrInvalidRetryMaxBackoff
	}

	if c.Work.MaxBackoff < c.Work.InitBackoff {
		return ErrRetryMaxBackoffTooSmall
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

// loadEnvInt reads key from the environment, parses it as an integer, and
// sets *dst if the value is in the range (min, max). Returns an error if
// the value cannot be parsed or falls outside the valid range.
func loadEnvInt[T ~int | ~int64](key string, dst *T, min, max T) error {
	s := os.Getenv(key)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid %s: %q", key, s)
	}
	tv := T(v)
	if tv <= min || tv > max {
		return fmt.Errorf("invalid %s: %d out of range [%d, %d]",
			key, tv, min+1, max)
	}
	*dst = tv
	return nil
}
