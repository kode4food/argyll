package archiver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/kode4food/timebox"
)

type (
	Writer struct {
		write  BucketWriteFunc
		prefix string
	}

	// BucketWriteFunc stores data at the provided key, replacing any existing
	// object for that key
	BucketWriteFunc func(ctx context.Context, key string, data []byte) error

	archiveObject struct {
		StreamID         string            `json:"stream_id"`
		AggregateID      string            `json:"aggregate_id"`
		SnapshotSequence int64             `json:"snapshot_sequence"`
		SnapshotData     json.RawMessage   `json:"snapshot_data"`
		Events           []json.RawMessage `json:"events"`
	}
)

var (
	ErrBucketRequired        = errors.New("bucket is required")
	ErrArchiveRecordRequired = errors.New("archive record is required")
)

func NewWriter(write BucketWriteFunc, prefix string) (*Writer, error) {
	if write == nil {
		return nil, ErrBucketRequired
	}
	return &Writer{
		write:  write,
		prefix: prefix,
	}, nil
}

func (w *Writer) Write(
	ctx context.Context, record *timebox.ArchiveRecord,
) error {
	if record == nil {
		return ErrArchiveRecordRequired
	}

	obj := archiveObject{
		StreamID:         record.StreamID,
		AggregateID:      record.AggregateID.Join(":"),
		SnapshotSequence: record.SnapshotSequence,
		SnapshotData:     normalizeRawMessage(record.SnapshotData),
		Events:           normalizeRawMessages(record.Events),
	}

	data, err := json.Marshal(&obj)
	if err != nil {
		return err
	}

	key := buildArchiveKey(w.prefix, record.AggregateID)
	return w.write(ctx, key, data)
}

func buildArchiveKey(prefix string, id timebox.AggregateID) string {
	if prefix == "" {
		return id.Join("/") + ".json"
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix + id.Join("/") + ".json"
}

func normalizeRawMessage(msg json.RawMessage) json.RawMessage {
	if len(strings.TrimSpace(string(msg))) == 0 {
		return nil
	}
	return msg
}

func normalizeRawMessages(msgs []json.RawMessage) []json.RawMessage {
	if len(msgs) == 0 {
		return nil
	}
	out := make([]json.RawMessage, 0, len(msgs))
	for _, msg := range msgs {
		if len(strings.TrimSpace(string(msg))) == 0 {
			continue
		}
		out = append(out, msg)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
