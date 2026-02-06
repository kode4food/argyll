package archiver_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"
	"gocloud.dev/blob"

	"github.com/kode4food/argyll/archiver"
	"github.com/kode4food/argyll/engine/pkg/events"

	_ "gocloud.dev/blob/memblob"
)

type (
	fakePoller struct {
		records chan *timebox.ArchiveRecord
		err     error
	}
)

const (
	testPollInterval = 10 * time.Millisecond
	testTimeout      = 2 * time.Second
)

func (p *fakePoller) PollArchive(
	ctx context.Context, timeout time.Duration, handler timebox.ArchiveHandler,
) error {
	if p.err != nil {
		return p.err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case rec := <-p.records:
		return handler(ctx, rec)
	case <-time.After(timeout):
		return nil
	}
}

func TestRunnerWritesArchiveRecord(t *testing.T) {
	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "mem://archiver-test")
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	w, err := archiver.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return b.WriteAll(ctx, key, data, nil)
		},
		"archived",
	)
	assert.NoError(t, err)

	p := &fakePoller{records: make(chan *timebox.ArchiveRecord, 1)}

	r, err := archiver.NewRunner(p, w, testPollInterval)
	assert.NoError(t, err)

	rec := &timebox.ArchiveRecord{
		StreamID:         "1-0",
		AggregateID:      events.FlowKey("abc123"),
		SnapshotSequence: 0,
		SnapshotData:     json.RawMessage{},
		Events: []json.RawMessage{
			json.RawMessage(`{"type":"flow_completed"}`),
		},
	}
	p.records <- rec

	err = r.RunOnce(ctx)
	assert.NoError(t, err)

	key := "archived/flow/abc123.json"
	raw, err := b.ReadAll(ctx, key)
	assert.NoError(t, err)

	var decoded map[string]any
	err = json.Unmarshal(raw, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, "1-0", decoded["stream_id"])
	assert.Equal(t, "flow:abc123", decoded["aggregate_id"])
}

func TestRunnerStopsOnPollerError(t *testing.T) {
	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "mem://archiver-test")
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	w, err := archiver.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return b.WriteAll(ctx, key, data, nil)
		},
		"",
	)
	assert.NoError(t, err)

	p := &fakePoller{err: errors.New("boom")}
	r, err := archiver.NewRunner(p, w, testPollInterval)
	assert.NoError(t, err)

	err = r.RunOnce(ctx)
	assert.Error(t, err)
}

func TestNewRunnerValidation(t *testing.T) {
	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "mem://archiver-test")
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	w, err := archiver.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return b.WriteAll(ctx, key, data, nil)
		},
		"",
	)
	assert.NoError(t, err)

	_, err = archiver.NewRunner(nil, w, testPollInterval)
	assert.Error(t, err)

	p := &fakePoller{records: make(chan *timebox.ArchiveRecord, 1)}
	_, err = archiver.NewRunner(p, nil, testPollInterval)
	assert.Error(t, err)

	_, err = archiver.NewRunner(p, w, 0)
	assert.Error(t, err)
}

func TestWriterRejectsNilRecord(t *testing.T) {
	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "mem://archiver-test")
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	w, err := archiver.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return b.WriteAll(ctx, key, data, nil)
		},
		"",
	)
	assert.NoError(t, err)

	err = w.Write(ctx, nil)
	assert.Error(t, err)
}

func TestWriterRejectsNilBucket(t *testing.T) {
	_, err := archiver.NewWriter(nil, "")
	assert.Error(t, err)
}

func TestWriterWritesWithNoPrefix(t *testing.T) {
	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "mem://archiver-test")
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	w, err := archiver.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return b.WriteAll(ctx, key, data, nil)
		},
		"",
	)
	assert.NoError(t, err)

	rec := &timebox.ArchiveRecord{
		StreamID:         "1-0",
		AggregateID:      events.FlowKey("abc123"),
		SnapshotSequence: 0,
		Events: []json.RawMessage{
			json.RawMessage(`{"type":"flow_completed"}`),
		},
	}

	err = w.Write(ctx, rec)
	assert.NoError(t, err)

	_, err = b.ReadAll(ctx, "flow/abc123.json")
	assert.NoError(t, err)
}

func TestWriterTrailingSlash(t *testing.T) {
	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "mem://archiver-test")
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	w, err := archiver.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return b.WriteAll(ctx, key, data, nil)
		},
		"archived/",
	)
	assert.NoError(t, err)

	rec := &timebox.ArchiveRecord{
		StreamID:         "1-0",
		AggregateID:      events.FlowKey("abc123"),
		SnapshotSequence: 0,
		Events: []json.RawMessage{
			json.RawMessage(`{"type":"flow_completed"}`),
		},
	}

	err = w.Write(ctx, rec)
	assert.NoError(t, err)

	_, err = b.ReadAll(ctx, "archived/flow/abc123.json")
	assert.NoError(t, err)
}

func TestWriterFiltersEmptyEvents(t *testing.T) {
	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "mem://archiver-test")
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	w, err := archiver.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return b.WriteAll(ctx, key, data, nil)
		},
		"archived",
	)
	assert.NoError(t, err)

	rec := &timebox.ArchiveRecord{
		StreamID:         "1-0",
		AggregateID:      events.FlowKey("abc123"),
		SnapshotSequence: 0,
		SnapshotData:     json.RawMessage(" "),
		Events: []json.RawMessage{
			json.RawMessage(""),
			json.RawMessage("   "),
			json.RawMessage(`{"type":"flow_completed"}`),
		},
	}

	err = w.Write(ctx, rec)
	assert.NoError(t, err)

	raw, err := b.ReadAll(ctx, "archived/flow/abc123.json")
	assert.NoError(t, err)

	var decoded map[string]any
	err = json.Unmarshal(raw, &decoded)
	assert.NoError(t, err)

	evs, ok := decoded["events"].([]any)
	assert.True(t, ok)
	assert.Len(t, evs, 1)
}

func TestWriterInvalidSnapshot(t *testing.T) {
	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "mem://archiver-test")
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	w, err := archiver.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return b.WriteAll(ctx, key, data, nil)
		},
		"archived",
	)
	assert.NoError(t, err)

	rec := &timebox.ArchiveRecord{
		StreamID:         "1-0",
		AggregateID:      events.FlowKey("abc123"),
		SnapshotSequence: 0,
		SnapshotData:     json.RawMessage("{"),
		Events: []json.RawMessage{
			json.RawMessage(`{"type":"flow_completed"}`),
		},
	}

	err = w.Write(ctx, rec)
	assert.Error(t, err)
}

func TestRunnerCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	b, err := blob.OpenBucket(context.Background(), "mem://archiver-test")
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	w, err := archiver.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return b.WriteAll(ctx, key, data, nil)
		},
		"",
	)
	assert.NoError(t, err)

	p := &fakePoller{records: make(chan *timebox.ArchiveRecord)}
	r, err := archiver.NewRunner(p, w, testPollInterval)
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
	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "mem://archiver-test")
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	w, err := archiver.NewWriter(
		func(ctx context.Context, key string, data []byte) error {
			return b.WriteAll(ctx, key, data, nil)
		},
		"",
	)
	assert.NoError(t, err)

	p := &fakePoller{err: context.Canceled}
	r, err := archiver.NewRunner(p, w, testPollInterval)
	assert.NoError(t, err)

	err = r.Run(ctx)
	assert.NoError(t, err)
}

func TestSetupLoggingDoesNotPanic(t *testing.T) {
	archiver.SetupLogging("debug")
	archiver.SetupLogging("not-a-level")
}
