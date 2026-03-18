package archive_test

import (
	"context"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/archive"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type fakeArchiver struct {
	records chan *timebox.ArchiveRecord
	err     error
}

const (
	testPollInterval = 10 * time.Millisecond
	testTimeout      = 2 * time.Second
)

func (a *fakeArchiver) Archive(timebox.AggregateID) error {
	return nil
}

func (a *fakeArchiver) ConsumeArchive(
	ctx context.Context, handler timebox.ArchiveHandler,
) error {
	if a.err != nil {
		return a.err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case rec := <-a.records:
		return handler(ctx, rec)
	}
}

func TestRunnerHandlesArchiveRecord(t *testing.T) {
	ctx := t.Context()
	a := &fakeArchiver{records: make(chan *timebox.ArchiveRecord, 1)}

	var got *timebox.ArchiveRecord
	r, err := archive.NewRunner(a, testPollInterval,
		func(_ context.Context, rec *timebox.ArchiveRecord) error {
			got = rec
			return nil
		},
	)
	assert.NoError(t, err)

	rec := &timebox.ArchiveRecord{
		StreamID:    "1-0",
		AggregateID: events.FlowKey("abc123"),
	}
	a.records <- rec

	err = r.RunOnce(ctx)
	assert.NoError(t, err)
	assert.Equal(t, rec, got)
}

func TestRunnerStopsOnPollerError(t *testing.T) {
	ctx := t.Context()
	a := &fakeArchiver{err: assert.AnError}

	r, err := archive.NewRunner(a, testPollInterval,
		func(context.Context, *timebox.ArchiveRecord) error {
			return nil
		},
	)
	assert.NoError(t, err)

	err = r.RunOnce(ctx)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestNewRunnerValidation(t *testing.T) {
	a := &fakeArchiver{records: make(chan *timebox.ArchiveRecord, 1)}
	handler := func(context.Context, *timebox.ArchiveRecord) error {
		return nil
	}

	_, err := archive.NewRunner(nil, testPollInterval, handler)
	assert.ErrorIs(t, err, archive.ErrArchiverRequired)

	_, err = archive.NewRunner(a, testPollInterval, nil)
	assert.ErrorIs(t, err, archive.ErrArchiveHandlerRequired)

	_, err = archive.NewRunner(a, 0, handler)
	assert.ErrorIs(t, err, archive.ErrPollIntervalInvalid)
}

func TestRunnerCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	a := &fakeArchiver{records: make(chan *timebox.ArchiveRecord)}
	r, err := archive.NewRunner(a, testPollInterval,
		func(context.Context, *timebox.ArchiveRecord) error {
			return nil
		},
	)
	assert.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		done <- r.Run(ctx)
	}()

	var runErr error
	ok := assert.Eventually(t, func() bool {
		select {
		case runErr = <-done:
			return true
		default:
			return false
		}
	}, testTimeout, testPollInterval)
	if ok {
		assert.NoError(t, runErr)
	}
}

func TestRunnerContextCanceledSuccess(t *testing.T) {
	r, err := archive.NewRunner(
		&fakeArchiver{err: context.Canceled},
		testPollInterval,
		func(context.Context, *timebox.ArchiveRecord) error {
			return nil
		},
	)
	assert.NoError(t, err)

	err = r.Run(t.Context())
	assert.NoError(t, err)
}

func TestSetupLoggingDoesNotPanic(t *testing.T) {
	archive.SetupLogging("debug")
	archive.SetupLogging("not-a-level")
}
