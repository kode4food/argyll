package hibernate

import (
	"context"
	"encoding/json"

	"github.com/kode4food/timebox"
	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"

	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

// BlobHibernator implements timebox.Hibernator using gocloud.dev/blob,
// supporting S3, GCS, Azure Blob Storage, and S3-compatible stores
type BlobHibernator struct {
	bucket *blob.Bucket
	prefix string
}

var _ timebox.Hibernator = (*BlobHibernator)(nil)

func NewBlobHibernator(
	ctx context.Context, bucketURL, prefix string,
) (*BlobHibernator, error) {
	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		return nil, err
	}
	return &BlobHibernator{bucket: bucket, prefix: prefix}, nil
}

func (h *BlobHibernator) Get(
	ctx context.Context, id timebox.AggregateID,
) (*timebox.HibernateRecord, error) {
	key := h.keyFor(id)
	data, err := h.bucket.ReadAll(ctx, key)
	if err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			return nil, timebox.ErrHibernateNotFound
		}
		return nil, err
	}

	var record timebox.HibernateRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (h *BlobHibernator) Put(
	ctx context.Context, id timebox.AggregateID, rec *timebox.HibernateRecord,
) error {
	key := h.keyFor(id)
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return h.bucket.WriteAll(ctx, key, data, nil)
}

func (h *BlobHibernator) Delete(
	ctx context.Context, id timebox.AggregateID,
) error {
	key := h.keyFor(id)
	err := h.bucket.Delete(ctx, key)
	if err != nil && gcerrors.Code(err) == gcerrors.NotFound {
		return nil
	}
	return err
}

func (h *BlobHibernator) Close() error {
	return h.bucket.Close()
}

func (h *BlobHibernator) keyFor(id timebox.AggregateID) string {
	return h.prefix + id.Join("/") + ".json"
}
