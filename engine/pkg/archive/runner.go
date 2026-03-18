package archive

import (
	"context"
	"errors"
	"time"

	"github.com/kode4food/timebox"
)

type (
	Runner struct {
		archiver     timebox.Archiver
		handler      timebox.ArchiveHandler
		pollInterval time.Duration
	}
)

var (
	ErrArchiverRequired       = errors.New("archiver is required")
	ErrArchiveHandlerRequired = errors.New("archive handler is required")
)

func NewRunner(
	archiver timebox.Archiver, pollInterval time.Duration,
	handler timebox.ArchiveHandler,
) (*Runner, error) {
	if archiver == nil {
		return nil, ErrArchiverRequired
	}
	if handler == nil {
		return nil, ErrArchiveHandlerRequired
	}
	if pollInterval <= 0 {
		return nil, ErrPollIntervalInvalid
	}
	return &Runner{
		archiver:     archiver,
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
	pollCtx, cancel := context.WithTimeout(ctx, r.pollInterval)
	defer cancel()

	err := r.archiver.ConsumeArchive(pollCtx, r.handler)
	if errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}
