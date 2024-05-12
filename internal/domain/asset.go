package domain

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const UnknownMediaTag = "__unknown__"

type AssetRepository interface {
	Init(ctx context.Context) error
	Get(ctx context.Context, uuid uuid.UUID) (Asset, io.ReadSeeker, error)
	Put(ctx context.Context, asset Asset, body io.ReadSeeker) (Asset, error)
	Delete(ctx context.Context, uuid uuid.UUID) error
}

type AssetUsecase interface {
	Create(ctx context.Context, author steamid.SteamID, bucket string, fileName string, content io.ReadSeeker) (Asset, error)
	Get(ctx context.Context, assetID uuid.UUID) (Asset, io.ReadSeeker, error)
	Delete(ctx context.Context, assetID uuid.UUID) error
}

type UserUploadedFile struct {
	Content string `json:"content"`
	Name    string `json:"name"`
	Mime    string `json:"mime"`
	Size    int64  `json:"size"`
}

type Asset struct {
	AssetID   uuid.UUID       `json:"asset_id"`
	Hash      []byte          `json:"-"` // 32 bytes
	AuthorID  steamid.SteamID `json:"author_id"`
	Bucket    string          `json:"bucket"`
	MimeType  string          `json:"mime_type"`
	Name      string          `json:"name"`
	Size      int64           `json:"size"`
	IsPrivate bool            `json:"is_private"`
	LocalPath string          `json:"-"`
	CreatedOn time.Time       `json:"created_on"`
	UpdatedOn time.Time       `json:"updated_on"`
}

func (a Asset) HashString() string {
	return fmt.Sprintf("%x", a.Hash)
}
