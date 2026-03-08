package cmd_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/archiver/internal/cmd"
	"github.com/kode4food/argyll/engine/pkg/archive"
)

func TestRunError(t *testing.T) {
	redisServer, err := miniredis.Run()
	assert.NoError(t, err)
	defer redisServer.Close()

	cfg, err := archive.LoadFromEnv()
	assert.NoError(t, err)

	cfg.FlowStore.Addr = redisServer.Addr()
	cfg.MemoryCheckInterval = 0

	err = cmd.Run(cfg, func(context.Context, *timebox.ArchiveRecord) error {
		return nil
	})
	assert.ErrorIs(t, err, archive.ErrMemoryCheckIntervalInvalid)
}
