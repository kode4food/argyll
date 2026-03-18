package archive

import (
	"context"
	"errors"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/redis"
	goredis "github.com/redis/go-redis/v9"
)

// Run starts the archive sweep loop and archive stream consumer using the
// flow-store configuration in cfg and invokes handler for each archived record
func Run(
	ctx context.Context, cfg Config, handler timebox.ArchiveHandler,
) error {
	SetupLogging(cfg.LogLevel)

	store, err := redis.NewStore(cfg.FlowStore)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	redisClient := goredis.NewClient(&goredis.Options{
		Addr:     cfg.FlowStore.Addr,
		Password: cfg.FlowStore.Password,
		DB:       cfg.FlowStore.DB,
	})
	defer func() { _ = redisClient.Close() }()

	arch, err := NewArchiver(store, redisClient, cfg)
	if err != nil {
		return err
	}

	runner, err := NewRunner(store, cfg.PollInterval, handler)
	if err != nil {
		return err
	}

	archErrCh := make(chan error, 1)
	go func() {
		archErrCh <- arch.Run(ctx)
	}()

	runnerErr := runner.Run(ctx)
	archErr := <-archErrCh

	if runnerErr != nil && !errors.Is(runnerErr, context.Canceled) {
		return runnerErr
	}
	if archErr != nil && !errors.Is(archErr, context.Canceled) {
		return archErr
	}
	return nil
}
