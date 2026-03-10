package archive

import (
	"context"
	"errors"

	"github.com/kode4food/timebox"
	"github.com/redis/go-redis/v9"

	"github.com/kode4food/argyll/engine/internal/config"
)

// Run starts the archive sweep loop and archive stream consumer using the
// flow-store configuration in cfg and invokes handler for each archived record
func Run(
	ctx context.Context, cfg Config, handler timebox.ArchiveHandler,
) error {
	SetupLogging(cfg.LogLevel)

	tb, err := timebox.NewTimebox(config.DefaultTimebox())
	if err != nil {
		return err
	}
	defer func() { _ = tb.Close() }()

	store, err := tb.NewStore(cfg.FlowStore, timebox.Config{
		Archiving: true,
	})
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.FlowStore.Redis.Addr,
		Password: cfg.FlowStore.Redis.Password,
		DB:       cfg.FlowStore.Redis.DB,
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
