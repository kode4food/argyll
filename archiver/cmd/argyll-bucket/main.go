package main

import (
	"context"
	"fmt"
	"os"

	"gocloud.dev/blob"

	"github.com/kode4food/argyll/archiver/internal/cmd"
	"github.com/kode4food/argyll/archiver/internal/writer"
	"github.com/kode4food/argyll/engine/pkg/archive"

	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

func main() {
	cfg, err := archive.LoadFromEnv()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	bucketCfg, err := loadBucketConfig()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()
	bucket, err := blob.OpenBucket(ctx, bucketCfg.BucketURL)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = bucket.Close() }()

	w, err := writer.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return bucket.WriteAll(ctx, key, data, nil)
		},
		bucketCfg.Prefix,
	)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := cmd.Run(cfg, w.Write); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
