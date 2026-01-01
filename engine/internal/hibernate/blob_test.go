package hibernate_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/hibernate"

	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/memblob"
)

func TestBlobHibernator(t *testing.T) {
	ctx := context.Background()

	h, err := hibernate.NewBlobHibernator(ctx, "mem://", "test/")
	assert.NoError(t, err)
	defer h.Close()

	id := timebox.NewAggregateID("flow", "wf-123")

	t.Run("Get returns not found for missing aggregate", func(t *testing.T) {
		_, err := h.Get(ctx, id)
		assert.ErrorIs(t, err, timebox.ErrHibernateNotFound)
	})

	t.Run("Put and Get round-trip", func(t *testing.T) {
		record := &timebox.HibernateRecord{
			Events: []json.RawMessage{
				json.RawMessage(`{"type":"flow_started"}`),
				json.RawMessage(`{"type":"step_completed"}`),
			},
			Snapshots: map[string]timebox.SnapshotRecord{
				"flow": {
					Data:     json.RawMessage(`{"id":"wf-123"}`),
					Sequence: 5,
				},
			},
		}

		err := h.Put(ctx, id, record)
		assert.NoError(t, err)

		got, err := h.Get(ctx, id)
		assert.NoError(t, err)

		assert.Len(t, got.Events, 2)
		assert.Contains(t, string(got.Events[0]), "flow_started")
		assert.Contains(t, string(got.Events[1]), "step_completed")
		assert.Equal(t, int64(5), got.Snapshots["flow"].Sequence)
	})

	t.Run("Delete removes aggregate", func(t *testing.T) {
		err := h.Delete(ctx, id)
		assert.NoError(t, err)

		_, err = h.Get(ctx, id)
		assert.ErrorIs(t, err, timebox.ErrHibernateNotFound)
	})

	t.Run("Delete on missing aggregate succeeds", func(t *testing.T) {
		missingID := timebox.NewAggregateID("flow", "nonexistent")
		err := h.Delete(ctx, missingID)
		assert.NoError(t, err)
	})
}

func TestBlobHibernatorKeyFormat(t *testing.T) {
	ctx := context.Background()

	h, err := hibernate.NewBlobHibernator(ctx, "mem://", "archived/")
	assert.NoError(t, err)
	defer h.Close()

	record := &timebox.HibernateRecord{
		Events: []json.RawMessage{json.RawMessage(`{}`)},
	}

	id := timebox.NewAggregateID("flow", "my-flow-id")
	err = h.Put(ctx, id, record)
	assert.NoError(t, err)

	got, err := h.Get(ctx, id)
	assert.NoError(t, err)
	assert.NotNil(t, got)
}

func TestBlobHibernatorFileURL(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	h, err := hibernate.NewBlobHibernator(ctx, "file://"+tmpDir, "")
	assert.NoError(t, err)
	defer h.Close()

	id := timebox.NewAggregateID("flow", "file-test")
	record := &timebox.HibernateRecord{
		Events: []json.RawMessage{json.RawMessage(`{"test":true}`)},
	}

	err = h.Put(ctx, id, record)
	assert.NoError(t, err)

	got, err := h.Get(ctx, id)
	assert.NoError(t, err)
	assert.Len(t, got.Events, 1)
}
