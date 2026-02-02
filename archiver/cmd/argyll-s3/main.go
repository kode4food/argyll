package main

import (
	"context"
	"fmt"
	"os"

	"gocloud.dev/blob"

	"github.com/kode4food/argyll/archiver"
	"github.com/kode4food/argyll/archiver/internal/cmd"

	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

func main() {
	cfg, err := archiver.LoadFromEnv()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	s3Cfg, err := loadS3Config()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()
	bucket, err := blob.OpenBucket(ctx, s3Cfg.BucketURL)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = bucket.Close() }()

	writer, err := archiver.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return bucket.WriteAll(ctx, key, data, nil)
		},
		s3Cfg.Prefix,
	)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := cmd.Run(cfg, writer, s3Cfg.PollInterval); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
