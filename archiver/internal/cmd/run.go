package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/archive"
)

func Run(cfg archive.Config, handler timebox.ArchiveHandler) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)

	go func() {
		<-stop
		cancel()
	}()

	return archive.Run(ctx, cfg, handler)
}
