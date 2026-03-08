package main

import (
	"errors"
	"os"
)

type s3Config struct {
	BucketURL string
	Prefix    string
}

var (
	ErrBucketURLRequired = errors.New("ARCHIVE_BUCKET_URL is required")
)

func loadS3Config() (s3Config, error) {
	cfg := s3Config{}

	if bucketURL := os.Getenv("ARCHIVE_BUCKET_URL"); bucketURL != "" {
		cfg.BucketURL = bucketURL
	}
	if prefix := os.Getenv("ARCHIVE_PREFIX"); prefix != "" {
		cfg.Prefix = prefix
	}
	if cfg.BucketURL == "" {
		return s3Config{}, ErrBucketURLRequired
	}

	return cfg, nil
}
