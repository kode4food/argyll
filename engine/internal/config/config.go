package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/raft"

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

	// Raft-backed persistence
	Raft raft.Config

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

	DefaultSnapshotWorkers   = 4
	DefaultSnapshotQueueSize = 1000
	DefaultTimeboxCacheSize  = 4096
	DefaultMemoCacheSize     = 16384
	DefaultRaftNodeID        = "argyll-1"
	DefaultRaftAddress       = "127.0.0.1:9701"
	DefaultRaftDataDirName   = "argyll-raft"

	DefaultRetryMaxRetries  = 10
	DefaultRetryInitBackoff = 1000
	DefaultMaxRetryBackoff  = 60000
	DefaultRetryBackoffType = api.BackoffTypeExponential

	MaxTimeboxCacheSize = 1_000_000
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
	ErrInvalidRaftServers      = errors.New("invalid RAFT_SERVERS")
)

// NewDefaultConfig creates a configuration with sensible defaults for all
// engine settings, stores, and retry behavior
func NewDefaultConfig() *Config {
	return &Config{
		APIPort:        DefaultAPIPort,
		APIHost:        DefaultAPIHost,
		WebhookBaseURL: "http://localhost:8080",
		Raft:           DefaultRaftConfig(),
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

func DefaultRaftConfig() raft.Config {
	cfg := raft.DefaultConfig().With(raft.Config{
		LocalID: DefaultRaftNodeID,
		Address: DefaultRaftAddress,
		DataDir: defaultRaftDataDir(DefaultRaftNodeID),
		Timebox: DefaultTimebox(),
	})
	cfg.Servers = defaultRaftServers(cfg)
	return cfg
}

// DefaultTimebox returns the top-level Timebox defaults Argyll expects
func DefaultTimebox() timebox.Config {
	return timebox.DefaultConfig().With(timebox.Config{
		Snapshot: timebox.SnapshotConfig{
			Workers:      true,
			WorkerCount:  DefaultSnapshotWorkers,
			MaxQueueSize: DefaultSnapshotQueueSize,
		},
		CacheSize: DefaultTimeboxCacheSize,
		Indexer:   events.FlowIndexer,
	})
}

// LoadFromEnv populates configuration values from environment variables
// Returns an error if any env var cannot be parsed
func (c *Config) LoadFromEnv() error {
	raftDataDirSet := false

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
	if raftNodeID := os.Getenv("RAFT_NODE_ID"); raftNodeID != "" {
		c.Raft.LocalID = raftNodeID
	}
	if raftAddress := os.Getenv("RAFT_ADDRESS"); raftAddress != "" {
		c.Raft.Address = raftAddress
	}
	if raftDataDir := os.Getenv("RAFT_DATA_DIR"); raftDataDir != "" {
		c.Raft.DataDir = raftDataDir
		raftDataDirSet = true
	}

	if err := loadEnvInt("API_PORT", &c.APIPort, 0, MaxTCPPort); err != nil {
		return err
	}

	if err := loadEnvInt(
		"TIMEBOX_CACHE_SIZE",
		&c.Raft.Timebox.CacheSize,
		0,
		MaxTimeboxCacheSize,
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
	if !raftDataDirSet {
		c.Raft.DataDir = defaultRaftDataDir(c.Raft.LocalID)
	}
	if raftServers := os.Getenv("RAFT_SERVERS"); raftServers != "" {
		srvs, err := parseRaftServers(raftServers)
		if err != nil {
			return err
		}
		c.Raft.Servers = srvs
	} else {
		c.Raft.Servers = defaultRaftServers(c.Raft)
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
		return fmt.Errorf("%w: %s", ErrInvalidRetryBackoffType,
			c.Work.BackoffType)
	}
	return c.Raft.Validate()
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

func defaultRaftServers(cfg raft.Config) []raft.Server {
	return []raft.Server{cfg.LocalServer()}
}

func defaultRaftDataDir(localID string) string {
	return filepath.Join(
		os.TempDir(), DefaultRaftDataDirName, localID,
	)
}

func parseRaftServers(spec string) ([]raft.Server, error) {
	parts := strings.Split(spec, ",")
	res := make([]raft.Server, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, addr, ok := strings.Cut(part, "=")
		id = strings.TrimSpace(id)
		addr = strings.TrimSpace(addr)
		if !ok || id == "" || addr == "" {
			return nil, fmt.Errorf("%w: %q", ErrInvalidRaftServers, part)
		}
		res = append(res, raft.Server{
			ID:      id,
			Address: addr,
		})
	}
	return res, nil
}
