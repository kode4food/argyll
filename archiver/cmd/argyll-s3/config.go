package main

import (
	"errors"
	"os"
	"time"
)

type s3Config struct {
	BucketURL    string
	Prefix       string
	PollInterval time.Duration
}

const defaultPollInterval = 500 * time.Millisecond

var (
	errBucketURLRequired = errors.New("ARCHIVE_BUCKET_URL is required")
)

func loadS3Config() (s3Config, error) {
	cfg := s3Config{
		PollInterval: defaultPollInterval,
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

	if cfg.BucketURL == "" {
		return s3Config{}, errBucketURLRequired
	}
	return cfg, nil
}
