package archive

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/config"
)

// Config configures the archiver runtime behavior
type Config struct {
	FlowStore           timebox.Config
	MemoryPercent       float64
	MaxAge              time.Duration
	MemoryCheckInterval time.Duration
	PollInterval        time.Duration
	SweepInterval       time.Duration
	LeaseTimeout        time.Duration
	PressureBatchSize   int
	SweepBatchSize      int
	LogLevel            string
}

const (
	DefaultMemoryPercent       = 80.0
	DefaultMaxAge              = 24 * time.Hour
	DefaultMemoryCheckInterval = 5 * time.Second
	DefaultPollInterval        = 500 * time.Millisecond
	DefaultSweepInterval       = 1 * time.Hour
	DefaultLeaseTimeout        = 15 * time.Minute
	DefaultPressureBatchSize   = 10
	DefaultSweepBatchSize      = 100

	defaultLogLevel = "info"
)

var (
	ErrMemoryCheckIntervalInvalid = errors.New(
		"ARCHIVE_MEMORY_CHECK_INTERVAL must be positive",
	)
	ErrPollIntervalInvalid = errors.New(
		"ARCHIVE_POLL_INTERVAL must be positive",
	)
	ErrSweepIntervalInvalid = errors.New(
		"ARCHIVE_SWEEP_INTERVAL must be positive",
	)
	ErrLeaseTimeoutInvalid = errors.New(
		"ARCHIVE_LEASE_TIMEOUT must be positive",
	)
	ErrPressureBatchInvalid = errors.New(
		"ARCHIVE_PRESSURE_BATCH must be positive",
	)
	ErrSweepBatchInvalid = errors.New(
		"ARCHIVE_SWEEP_BATCH must be positive",
	)
)

func LoadFromEnv() (Config, error) {
	flowStore := config.NewDefaultConfig().FlowStore

	cfg := Config{
		FlowStore:           flowStore,
		MemoryPercent:       DefaultMemoryPercent,
		MaxAge:              DefaultMaxAge,
		MemoryCheckInterval: DefaultMemoryCheckInterval,
		PollInterval:        DefaultPollInterval,
		SweepInterval:       DefaultSweepInterval,
		LeaseTimeout:        DefaultLeaseTimeout,
		PressureBatchSize:   DefaultPressureBatchSize,
		SweepBatchSize:      DefaultSweepBatchSize,
		LogLevel:            defaultLogLevel,
	}
	config.LoadStoreConfigFromEnv(&cfg.FlowStore, "PARTITION")

	if pct := os.Getenv("ARCHIVE_MEMORY_PERCENT"); pct != "" {
		if f, err := strconv.ParseFloat(pct, 64); err == nil {
			cfg.MemoryPercent = f
		}
	}
	if maxAge := os.Getenv("ARCHIVE_MAX_AGE"); maxAge != "" {
		if d, err := time.ParseDuration(maxAge); err == nil {
			cfg.MaxAge = d
		}
	}
	if interval := os.Getenv("ARCHIVE_MEMORY_CHECK_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			cfg.MemoryCheckInterval = d
		}
	}
	if interval := os.Getenv("ARCHIVE_POLL_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			cfg.PollInterval = d
		}
	}
	if interval := os.Getenv("ARCHIVE_SWEEP_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			cfg.SweepInterval = d
		}
	}
	if timeout := os.Getenv("ARCHIVE_LEASE_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			cfg.LeaseTimeout = d
		}
	}
	if val := os.Getenv("ARCHIVE_PRESSURE_BATCH"); val != "" {
		if size, err := strconv.Atoi(val); err == nil {
			cfg.PressureBatchSize = size
		}
	}
	if val := os.Getenv("ARCHIVE_SWEEP_BATCH"); val != "" {
		if size, err := strconv.Atoi(val); err == nil {
			cfg.SweepBatchSize = size
		}
	}
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if err := c.validateArchiver(); err != nil {
		return err
	}
	return c.validateRunner()
}

func (c Config) validateArchiver() error {
	if c.MemoryCheckInterval <= 0 {
		return ErrMemoryCheckIntervalInvalid
	}
	if c.SweepInterval <= 0 {
		return ErrSweepIntervalInvalid
	}
	if c.LeaseTimeout <= 0 {
		return ErrLeaseTimeoutInvalid
	}
	if c.PressureBatchSize <= 0 {
		return ErrPressureBatchInvalid
	}
	if c.SweepBatchSize <= 0 {
		return ErrSweepBatchInvalid
	}
	return nil
}

func (c Config) validateRunner() error {
	if c.PollInterval <= 0 {
		return ErrPollIntervalInvalid
	}
	return nil
}
