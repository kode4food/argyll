package archiver

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/kode4food/timebox"
)

// Config configures the archiver runtime behavior
type Config struct {
	EngineStore         timebox.StoreConfig
	FlowStore           timebox.StoreConfig
	MemoryPercent       float64
	MaxAge              time.Duration
	MemoryCheckInterval time.Duration
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
	cfg := Config{
		EngineStore:         timebox.DefaultStoreConfig(),
		FlowStore:           timebox.DefaultStoreConfig(),
		MemoryPercent:       DefaultMemoryPercent,
		MaxAge:              DefaultMaxAge,
		MemoryCheckInterval: DefaultMemoryCheckInterval,
		SweepInterval:       DefaultSweepInterval,
		LeaseTimeout:        DefaultLeaseTimeout,
		PressureBatchSize:   DefaultPressureBatchSize,
		SweepBatchSize:      DefaultSweepBatchSize,
		LogLevel:            defaultLogLevel,
	}

	loadStoreConfigFromEnv(&cfg.EngineStore, "ENGINE")
	loadStoreConfigFromEnv(&cfg.FlowStore, "FLOW")

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

func loadStoreConfigFromEnv(s *timebox.StoreConfig, prefix string) {
	if addr := os.Getenv(prefix + "_REDIS_ADDR"); addr != "" {
		s.Addr = addr
	}
	if password := os.Getenv(prefix + "_REDIS_PASSWORD"); password != "" {
		s.Password = password
	}
	if dbStr := os.Getenv(prefix + "_REDIS_DB"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
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
