package cmd

import (
	"os"
	"time"
)

type Config struct {
	PollInterval time.Duration
}

const DefaultPollInterval = 500 * time.Millisecond

func LoadConfig(cfg *Config) {
	if cfg == nil {
		return
	}

	if cfg.PollInterval == 0 {
		cfg.PollInterval = DefaultPollInterval
	}

	if val := os.Getenv("ARCHIVE_POLL_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.PollInterval = d
		}
	}
}
