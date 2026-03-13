package archive_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/archive"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func TestRunCanceled(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	cfg := testRunConfig(redisServer.Addr())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err = archive.Run(ctx, cfg,
		func(context.Context, *timebox.ArchiveRecord) error {
			return nil
		},
	)
	assert.NoError(t, err)
}

func TestRunNoHandler(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	cfg := testRunConfig(redisServer.Addr())

	err = archive.Run(t.Context(), cfg, nil)
	assert.ErrorIs(t, err, archive.ErrArchiveHandlerRequired)
}

func TestRunBadConfig(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	cfg := testRunConfig(redisServer.Addr())
	cfg.MemoryCheckInterval = 0

	err = archive.Run(t.Context(), cfg,
		func(context.Context, *timebox.ArchiveRecord) error {
			return nil
		},
	)
	assert.ErrorIs(t, err, archive.ErrMemoryCheckIntervalInvalid)
}

func TestRunBadPollInterval(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	cfg := testRunConfig(redisServer.Addr())
	cfg.PollInterval = 0

	err = archive.Run(t.Context(), cfg,
		func(context.Context, *timebox.ArchiveRecord) error {
			return nil
		},
	)
	assert.ErrorIs(t, err, archive.ErrPollIntervalInvalid)
}

func TestRunArchives(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	flowStore := setupStore(t, redisServer.Addr())
	id := api.FlowID("flow-run")
	seedDeactivatedFlow(t, flowStore, id)

	cfg := testRunConfig(redisServer.Addr())
	cfg.MaxAge = 0

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)
	records := make(chan *timebox.ArchiveRecord, 1)
	go func() {
		done <- archive.Run(ctx, cfg,
			func(_ context.Context, rec *timebox.ArchiveRecord) error {
				records <- rec
				cancel()
				return nil
			},
		)
	}()

	var record *timebox.ArchiveRecord
	ok := assert.Eventually(t, func() bool {
		select {
		case record = <-records:
			return true
		default:
			return false
		}
	}, testTimeout, testPollInterval)
	if ok {
		assert.Equal(t, events.FlowKey(id), record.AggregateID)
	}

	assert.NoError(t, <-done)
}

func TestRunHandlerError(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	flowStore := setupStore(t, redisServer.Addr())
	id := api.FlowID("flow-run-error")
	seedDeactivatedFlow(t, flowStore, id)

	cfg := testRunConfig(redisServer.Addr())
	cfg.MaxAge = 0

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	err = archive.Run(ctx, cfg,
		func(context.Context, *timebox.ArchiveRecord) error {
			cancel()
			return assert.AnError
		},
	)
	assert.ErrorIs(t, err, assert.AnError)
}

func testRunConfig(addr string) archive.Config {
	cfg := archive.Config{
		FlowStore:           config.NewDefaultConfig().FlowStore,
		MemoryPercent:       archive.DefaultMemoryPercent,
		MaxAge:              archive.DefaultMaxAge,
		MemoryCheckInterval: 10 * time.Millisecond,
		PollInterval:        10 * time.Millisecond,
		SweepInterval:       10 * time.Millisecond,
		LeaseTimeout:        time.Second,
		PressureBatchSize:   1,
		SweepBatchSize:      1,
		LogLevel:            "debug",
	}
	cfg.FlowStore.Addr = addr
	cfg.FlowStore.Prefix = "partition"
	return cfg
}
