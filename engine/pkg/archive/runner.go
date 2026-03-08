package archive

import (
	"context"
	"errors"
	"time"

	"github.com/kode4food/timebox"
)

type (
	Runner struct {
		poller       Poller
		handler      timebox.ArchiveHandler
		pollInterval time.Duration
	}

	Poller interface {
		PollArchive(
			context.Context, time.Duration, timebox.ArchiveHandler,
		) error
	}
)

var (
	ErrArchivePollerRequired  = errors.New("archive poller is required")
	ErrArchiveHandlerRequired = errors.New("archive handler is required")
)

func NewRunner(
	poller Poller, pollInterval time.Duration, handler timebox.ArchiveHandler,
) (*Runner, error) {
	if poller == nil {
		return nil, ErrArchivePollerRequired
	}
	if handler == nil {
		return nil, ErrArchiveHandlerRequired
	}
	if pollInterval <= 0 {
		return nil, ErrPollIntervalInvalid
	}
	return &Runner{
		poller:       poller,
		handler:      handler,
		pollInterval: pollInterval,
	}, nil
}

func (r *Runner) Run(ctx context.Context) error {
	for ctx.Err() == nil {
		if err := r.RunOnce(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
	}
	return nil
}

func (r *Runner) RunOnce(ctx context.Context) error {
	return r.poller.PollArchive(ctx, r.pollInterval, r.handler)
}
