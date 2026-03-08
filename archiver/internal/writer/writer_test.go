package writer_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"
	"gocloud.dev/blob"

	"github.com/kode4food/argyll/archiver/internal/writer"
	"github.com/kode4food/argyll/engine/pkg/events"

	_ "gocloud.dev/blob/memblob"
)

func TestWriterRejectsNilRecord(t *testing.T) {
	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "mem://archiver-test")
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	w, err := writer.NewWriter(
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
	_, err := writer.NewWriter(nil, "")
	assert.Error(t, err)
}

func TestWriterWritesWithNoPrefix(t *testing.T) {
	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "mem://archiver-test")
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	w, err := writer.NewWriter(
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

	w, err := writer.NewWriter(
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

	w, err := writer.NewWriter(
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

	w, err := writer.NewWriter(
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
