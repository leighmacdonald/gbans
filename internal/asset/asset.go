package asset

import (
	"encoding/hex"
	"mime/multipart"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const UnknownMediaTag = "__unknown__"

type Bucket string

const (
	BucketDemo  Bucket = "demos"
	BucketMedia Bucket = "media"
)

type UserUploadedFile struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
	Name string                `form:"name"`
}

type Asset struct {
	AssetID   uuid.UUID       `json:"asset_id"`
	Hash      []byte          `json:"-"` // 32 bytes
	AuthorID  steamid.SteamID `json:"author_id"`
	Bucket    Bucket          `json:"bucket"`
	MimeType  string          `json:"mime_type"`
	Name      string          `json:"name"`
	Size      int64           `json:"size"`
	IsPrivate bool            `json:"is_private"`
	LocalPath string          `json:"-"`
	CreatedOn time.Time       `json:"created_on"`
	UpdatedOn time.Time       `json:"updated_on"`
}

func (a Asset) HashString() string {
	return hex.EncodeToString(a.Hash)
}
