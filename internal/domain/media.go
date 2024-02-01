package domain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type MediaRepository interface {
	SaveMedia(ctx context.Context, media *Media) error
	GetMediaByAssetID(ctx context.Context, uuid uuid.UUID, media *Media) error
	GetMediaByName(ctx context.Context, name string, media *Media) error
	GetMediaByID(ctx context.Context, mediaID int, media *Media) error
}

type MediaUsecase interface {
	Create(ctx context.Context, steamID steamid.SID64, name string, mimeType string, content []byte, mimeTypesAllowed []string) (*Media, error)
	GetMediaByAssetID(ctx context.Context, uuid uuid.UUID, media *Media) error
	GetMediaByName(ctx context.Context, name string, media *Media) error
	GetMediaByID(ctx context.Context, mediaID int, media *Media) error
}

func NewMedia(author steamid.SID64, name string, mime string, content []byte) (Media, error) {
	mType := mimetype.Detect(content)
	if !mType.Is(mime) && mime != unknownMediaTag {
		// Should never actually happen unless user is trying nefarious stuff.
		return Media{}, fmt.Errorf("%w: %s = %s", ErrInvalidMediaMimeType, mime, mType.String())
	}

	curTime := time.Now()

	return Media{
		AuthorID:  author,
		MimeType:  mType.String(),
		Name:      strings.ReplaceAll(name, " ", "_"),
		Size:      int64(len(content)),
		Contents:  content,
		Deleted:   false,
		CreatedOn: curTime,
		UpdatedOn: curTime,
		Asset:     Asset{},
	}, nil
}

type Media struct {
	MediaID   int           `json:"media_id"`
	AuthorID  steamid.SID64 `json:"author_id"`
	MimeType  string        `json:"mime_type"`
	Contents  []byte        `json:"-"`
	Name      string        `json:"name"`
	Size      int64         `json:"size"`
	Deleted   bool          `json:"deleted"`
	CreatedOn time.Time     `json:"created_on"`
	UpdatedOn time.Time     `json:"updated_on"`
	Asset     Asset         `json:"asset"`
}
