package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

const unknownMediaTag = "__unknown__"

var ErrInvalidMediaMimeType = errors.New("detected mimetype different than type provided")

type WikiRepository interface {
	GetWikiPageBySlug(ctx context.Context, slug string, page *wiki.Page) error
	DeleteWikiPageBySlug(ctx context.Context, slug string) error
	SaveWikiPage(ctx context.Context, page *wiki.Page) error
	SaveMedia(ctx context.Context, media *Media) error
	GetMediaByAssetID(ctx context.Context, uuid uuid.UUID, media *Media) error
	GetMediaByName(ctx context.Context, name string, media *Media) error
	GetMediaByID(ctx context.Context, mediaID int, media *Media) error
}

type WikiUsecase interface {
	GetWikiPageBySlug(ctx context.Context, slug string, page *wiki.Page) error
	DeleteWikiPageBySlug(ctx context.Context, slug string) error
	SaveWikiPage(ctx context.Context, page *wiki.Page) error
	SaveMedia(ctx context.Context, media *Media) error
	GetMediaByAssetID(ctx context.Context, uuid uuid.UUID, media *Media) error
	GetMediaByName(ctx context.Context, name string, media *Media) error
	GetMediaByID(ctx context.Context, mediaID int, media *Media) error
}

// TODO move media to separate pkg
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
