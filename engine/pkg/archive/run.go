package archive

import (
	"context"
	"errors"

	"github.com/kode4food/timebox"
	"github.com/redis/go-redis/v9"
)

// Run starts the archive sweep loop and archive stream consumer using the
// flow-store configuration in cfg and invokes handler for each archived record
func Run(
	ctx context.Context, cfg Config, handler timebox.ArchiveHandler,
) error {
	SetupLogging(cfg.LogLevel)

	tbCfg := timebox.DefaultConfig()
	tbCfg.Workers = false
	tb, err := timebox.NewTimebox(tbCfg)
	if err != nil {
		return err
	}
	defer func() { _ = tb.Close() }()

	storeCfg := cfg.FlowStore
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
