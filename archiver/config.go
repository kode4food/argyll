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
	FlowStore    timebox.StoreConfig
	BucketURL    string
	Prefix       string
	PollInterval time.Duration
	LogLevel     string
}

const (
	DefaultPollInterval = 500 * time.Millisecond

	defaultLogLevel = "info"
)

var (
	ErrBucketURLRequired = errors.New("ARCHIVE_BUCKET_URL is required")
)

func LoadFromEnv() (Config, error) {
	cfg := Config{
		FlowStore:    timebox.DefaultStoreConfig(),
		PollInterval: DefaultPollInterval,
		LogLevel:     defaultLogLevel,
	}

	if addr := os.Getenv("FLOW_REDIS_ADDR"); addr != "" {
		cfg.FlowStore.Addr = addr
	}
	if password := os.Getenv("FLOW_REDIS_PASSWORD"); password != "" {
		cfg.FlowStore.Password = password
	}
	if dbStr := os.Getenv("FLOW_REDIS_DB"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			cfg.FlowStore.DB = db
		}
	}
	if prefix := os.Getenv("FLOW_REDIS_PREFIX"); prefix != "" {
		cfg.FlowStore.Prefix = prefix
	}

	if bucketURL := os.Getenv("ARCHIVE_BUCKET_URL"); bucketURL != "" {
		cfg.BucketURL = bucketURL
	}
	if prefix := os.Getenv("ARCHIVE_PREFIX"); prefix != "" {
		cfg.Prefix = prefix
	}
	if val := os.Getenv("ARCHIVE_POLL_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.PollInterval = d
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
	if c.BucketURL == "" {
		return ErrBucketURLRequired
	}
	return nil
}
