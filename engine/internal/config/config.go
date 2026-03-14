package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/postgres"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

// Config holds configuration settings for the orchestrator
type Config struct {
	// API Server
	APIHost        string
	APIPort        int
	WebhookBaseURL string
	LogLevel       string

	// Stores & Archiving
	CatalogStore   postgres.Config
	PartitionStore postgres.Config
	FlowStore      postgres.Config

	// Work & Retry
	Work api.WorkConfig

	// Engine
	StepTimeout     int64
	MemoCacheSize   int
	ShutdownTimeout time.Duration
}

const (
	DefaultStepTimeout     = 30 * api.Second
	DefaultShutdownTimeout = 10 * time.Second

	DefaultAPIPort = 8080
	DefaultAPIHost = "0.0.0.0"
	MaxTCPPort     = 65535

	DefaultPostgresURL         = "postgres://localhost:5432/argyll?sslmode=disable"
	DefaultPostgresPrefix      = "argyll"
	DefaultPostgresMaxConns    = postgres.DefaultMaxConns
	DefaultSnapshotWorkers     = 4
	DefaultSnapshotQueueSize   = 1000
	DefaultSnapshotSaveTimeout = 30 * time.Second
	DefaultFlowCacheSize       = 4096
	DefaultMemoCacheSize       = 16384

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
	base := postgres.DefaultConfig().With(postgres.Config{
		Timebox:  DefaultTimebox(),
		URL:      DefaultPostgresURL,
		Prefix:   DefaultPostgresPrefix,
		MaxConns: DefaultPostgresMaxConns,
	})

	return &Config{
		APIPort:        DefaultAPIPort,
		APIHost:        DefaultAPIHost,
		WebhookBaseURL: "http://localhost:8080",
		CatalogStore: base.With(postgres.Config{
			Timebox: timebox.Config{
				Snapshot: timebox.SnapshotConfig{
					TrimEvents: true,
				},
			},
		}),
		PartitionStore: base.With(postgres.Config{
			Timebox: timebox.Config{
				Snapshot: timebox.SnapshotConfig{
					TrimEvents: true,
				},
			},
		}),
		FlowStore: base.With(postgres.Config{
			Timebox: timebox.Config{
				Indexer:   events.FlowIndexer,
				CacheSize: DefaultFlowCacheSize,
			},
		}),
		Work: api.WorkConfig{
			MaxRetries:  DefaultRetryMaxRetries,
			InitBackoff: DefaultRetryInitBackoff,
			MaxBackoff:  DefaultMaxRetryBackoff,
			BackoffType: DefaultRetryBackoffType,
		},
		StepTimeout:     DefaultStepTimeout,
		MemoCacheSize:   DefaultMemoCacheSize,
		ShutdownTimeout: DefaultShutdownTimeout,
		LogLevel:        "info",
	}
}

// DefaultTimebox returns the top-level Timebox defaults Argyll expects
func DefaultTimebox() timebox.Config {
	return timebox.DefaultConfig().With(timebox.Config{
		Snapshot: timebox.SnapshotConfig{
			Workers:      true,
			WorkerCount:  DefaultSnapshotWorkers,
			MaxQueueSize: DefaultSnapshotQueueSize,
			SaveTimeout:  DefaultSnapshotSaveTimeout,
		},
	})
}

// LoadFromEnv populates configuration values from environment variables
// Returns an error if any env var cannot be parsed
func (c *Config) LoadFromEnv() error {
	LoadStoreConfigFromEnv(&c.CatalogStore, "CATALOG")
	LoadStoreConfigFromEnv(&c.PartitionStore, "PARTITION")
	LoadStoreConfigFromEnv(&c.FlowStore, "FLOW")

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
		"FLOW_CACHE_SIZE", &c.FlowStore.Timebox.CacheSize, 0, MaxFlowCacheSize,
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

	if err := c.CatalogStore.Validate(); err != nil {
		return err
	}
	if err := c.PartitionStore.Validate(); err != nil {
		return err
	}
	if err := c.FlowStore.Validate(); err != nil {
		return err
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
		return fmt.Errorf("%w: %s", ErrInvalidRetryBackoffType,
			c.Work.BackoffType)
	}

	return nil
}

// LoadStoreConfigFromEnv loads Postgres store configuration from environment
// variables with the given prefix
func LoadStoreConfigFromEnv(s *postgres.Config, prefix string) {
	if url := os.Getenv(prefix + "_POSTGRES_URL"); url != "" {
		s.URL = url
	}
	if envPrefix := os.Getenv(prefix + "_POSTGRES_PREFIX"); envPrefix != "" {
		s.Prefix = envPrefix
	}
	if maxConns := os.Getenv(prefix + "_POSTGRES_MAX_CONNS"); maxConns != "" {
		if v, err := strconv.ParseInt(maxConns, 10, 32); err == nil && v > 0 {
			s.MaxConns = int32(v)
		}
	}
	if envCount := os.Getenv(prefix + "_SNAPSHOT_WORKERS"); envCount != "" {
		if wc, err := strconv.Atoi(envCount); err == nil && wc >= 0 {
			s.Timebox.Snapshot.WorkerCount = wc
		}
	}
}

// loadEnvInt reads key from the environment, parses it as an integer, and
// sets *dst if the value is in the range (min, max). Returns an error if
// the value cannot be parsed or falls outside the valid range
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
