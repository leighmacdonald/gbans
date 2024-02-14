package domain

import (
	"context"
	"io"

	"github.com/gofrs/uuid/v5"
)

type AssetRepository interface {
	GetAsset(ctx context.Context, uuid uuid.UUID) (Asset, error)
	CreateBucketIfNotExists(ctx context.Context, name string) error
	SaveAsset(ctx context.Context, asset *Asset) error
	DeleteAsset(ctx context.Context, asset *Asset) error
	Get(ctx context.Context, bucket string, name string) (io.Reader, error)
	Put(ctx context.Context, bucket string, name string, body io.Reader, size int64, contentType string) error
	Remove(ctx context.Context, bucket string, name string) error
	LinkObject(bucket string, name string) string
}

// TODO SaveAsset/DropAsset higher level funcs.
type AssetUsecase interface {
	GetAsset(ctx context.Context, uuid uuid.UUID) (Asset, io.Reader, error)
	SaveAsset(ctx context.Context, bucket string, asset *Asset, content []byte) error
	DropAsset(ctx context.Context, asset *Asset) error
}

type UserUploadedFile struct {
	Content string `json:"content"`
	Name    string `json:"name"`
	Mime    string `json:"mime"`
	Size    int64  `json:"size"`
}
