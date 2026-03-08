package main

import (
	"errors"
	"os"
)

type bucketConfig struct {
	BucketURL string
	Prefix    string
}

var (
	ErrBucketURLRequired = errors.New("ARCHIVE_BUCKET_URL is required")
)

func loadBucketConfig() (bucketConfig, error) {
	cfg := bucketConfig{}

	if bucketURL := os.Getenv("ARCHIVE_BUCKET_URL"); bucketURL != "" {
		cfg.BucketURL = bucketURL
	}
	if prefix := os.Getenv("ARCHIVE_PREFIX"); prefix != "" {
		cfg.Prefix = prefix
	}
	if cfg.BucketURL == "" {
		return bucketConfig{}, ErrBucketURLRequired
	}

	return cfg, nil
}
