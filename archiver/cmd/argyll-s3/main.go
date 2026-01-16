package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kode4food/timebox"
	"github.com/redis/go-redis/v9"
	"gocloud.dev/blob"

	"github.com/kode4food/argyll/archiver"

	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

func main() {
	cfg, err := archiver.LoadFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	s3Cfg, err := loadS3Config()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	archiver.SetupLogging(cfg.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)

	go func() {
		<-stop
		cancel()
	}()

	bucket, err := blob.OpenBucket(ctx, s3Cfg.BucketURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer bucket.Close()

	tbCfg := timebox.DefaultConfig()
	tbCfg.Workers = false
	tb, err := timebox.NewTimebox(tbCfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer tb.Close()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer engineStore.Close()

	storeCfg := cfg.FlowStore
	storeCfg.Archiving = true
	store, err := tb.NewStore(storeCfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer store.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     storeCfg.Addr,
		Password: storeCfg.Password,
		DB:       storeCfg.DB,
	})
	defer redisClient.Close()

	arch, err := archiver.NewArchiver(engineStore, store, redisClient, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	writer, err := archiver.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return bucket.WriteAll(ctx, key, data, nil)
		},
		s3Cfg.Prefix,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runner, err := archiver.NewRunner(store, writer, s3Cfg.PollInterval)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	go func() {
		if err := arch.Run(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			cancel()
		}
	}()

	if err := runner.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
