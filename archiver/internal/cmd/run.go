package cmd

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kode4food/timebox"
	"github.com/redis/go-redis/v9"

	"github.com/kode4food/argyll/archiver"
)

func Run(
	cfg archiver.Config, writer *archiver.Writer, pollInterval time.Duration,
) error {
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

	tbCfg := timebox.DefaultConfig()
	tbCfg.Workers = false
	tb, err := timebox.NewTimebox(tbCfg)
	if err != nil {
		return err
	}
	defer func() { _ = tb.Close() }()

	storeCfg := cfg.PartitionStore
	storeCfg.Archiving = true
	store, err := tb.NewStore(storeCfg)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     storeCfg.Addr,
		Password: storeCfg.Password,
		DB:       storeCfg.DB,
	})
	defer func() { _ = redisClient.Close() }()

	arch, err := archiver.NewArchiver(store, redisClient, cfg)
	if err != nil {
		return err
	}

	runner, err := archiver.NewRunner(store, writer, pollInterval)
	if err != nil {
		return err
	}

	archErrCh := make(chan error, 1)
	go func() {
		archErrCh <- arch.Run(ctx)
	}()

	runnerErr := runner.Run(ctx)
	cancel()
	archErr := <-archErrCh

	if runnerErr != nil && !errors.Is(runnerErr, context.Canceled) {
		return runnerErr
	}
	if archErr != nil && !errors.Is(archErr, context.Canceled) {
		return archErr
	}
	return nil
}
