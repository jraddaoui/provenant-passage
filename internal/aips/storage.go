// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
	"context"
	"errors"
	"io"

	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"
)

var ErrObjectNotFound = errors.New("object not found")

type (
	BlobReader interface {
		io.ReadCloser
		Size() int64
	}
	Store interface {
		NewWriter(context.Context, string) (io.WriteCloser, error)
		NewReader(context.Context, string) (BlobReader, error)
		Delete(context.Context, string) error
	}
)

type Bucket struct{ Bucket *blob.Bucket }

func (b Bucket) NewWriter(ctx context.Context, key string) (io.WriteCloser, error) {
	return b.Bucket.NewWriter(ctx, key, nil)
}

func (b Bucket) NewReader(ctx context.Context, key string) (BlobReader, error) {
	r, err := b.Bucket.NewReader(ctx, key, nil)
	if gcerrors.Code(err) == gcerrors.NotFound {
		return nil, ErrObjectNotFound
	}
	return r, err
}

func (b Bucket) Delete(ctx context.Context, key string) error {
	err := b.Bucket.Delete(ctx, key)
	if gcerrors.Code(err) == gcerrors.NotFound {
		return nil
	}
	return err
}
