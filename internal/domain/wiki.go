package domain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

const unknownMediaTag = "__unknown__"

type WikiRepository interface {
	GetWikiPageBySlug(ctx context.Context, slug string, page *Page) error
	DeleteWikiPageBySlug(ctx context.Context, slug string) error
	SaveWikiPage(ctx context.Context, page *Page) error
}

type WikiUsecase interface {
	GetWikiPageBySlug(ctx context.Context, slug string, page *Page) error
	DeleteWikiPageBySlug(ctx context.Context, slug string) error
	SaveWikiPage(ctx context.Context, page *Page) error
}

// RootSlug is the top-most (index) page of the wiki.
const RootSlug = "home"

type Page struct {
	Slug            string    `json:"slug"`
	BodyMD          string    `json:"body_md"`
	Revision        int       `json:"revision"`
	PermissionLevel Privilege `json:"permission_level"`
	CreatedOn       time.Time `json:"created_on"`
	UpdatedOn       time.Time `json:"updated_on"`
}

func (page *Page) NewRevision() Page {
	return Page{
		Slug:      page.Slug,
		BodyMD:    page.BodyMD,
		Revision:  page.Revision + 1,
		CreatedOn: page.CreatedOn,
		UpdatedOn: time.Now(),
	}
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
