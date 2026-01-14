package archiver

import (
	"context"
	"errors"
	"time"

	"github.com/kode4food/timebox"
)

type (
	Runner struct {
		poller       ArchivePoller
		writer       *Writer
		pollInterval time.Duration
	}

	ArchivePoller interface {
		PollArchive(
			context.Context, time.Duration, timebox.ArchiveHandler,
		) error
	}
)

var (
	ErrArchivePollerRequired = errors.New("archive poller is required")
	ErrArchiveWriterRequired = errors.New("archive writer is required")
	ErrPollIntervalInvalid   = errors.New("poll interval must be positive")
)

func NewRunner(
	poller ArchivePoller, writer *Writer, pollInterval time.Duration,
) (*Runner, error) {
	if poller == nil {
		return nil, ErrArchivePollerRequired
	}
	if writer == nil {
		return nil, ErrArchiveWriterRequired
	}
	if pollInterval <= 0 {
		return nil, ErrPollIntervalInvalid
	}
	return &Runner{
		poller:       poller,
		writer:       writer,
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
	return r.poller.PollArchive(ctx, r.pollInterval, r.writer.Write)
}
