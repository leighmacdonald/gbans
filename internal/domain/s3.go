package domain

import (
	"context"
	"io"
)

type S3Usecase interface {
	CreateBucketIfNotExists(ctx context.Context, name string) error
	Put(ctx context.Context, bucket string, name string, body io.Reader, size int64, contentType string) error
	Remove(ctx context.Context, bucket string, name string) error
	LinkObject(bucket string, name string) string
}
