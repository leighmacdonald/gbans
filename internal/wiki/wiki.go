package wiki

import (
	"context"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type WikiRepository interface {
	GetWikiPageBySlug(ctx context.Context, slug string) (Page, error)
	DeleteWikiPageBySlug(ctx context.Context, slug string) error
	SaveWikiPage(ctx context.Context, page *Page) error
}

type WikiUsecase interface {
	GetWikiPageBySlug(ctx context.Context, user domain.PersonInfo, slug string) (Page, error)
	DeleteWikiPageBySlug(ctx context.Context, slug string) error
	SaveWikiPage(ctx context.Context, user domain.PersonInfo, slug string, body string, level domain.Privilege) (Page, error)
}

// RootSlug is the top-most (index) page of the wiki.
const RootSlug = "home"

type Page struct {
	Slug            string           `json:"slug"`
	BodyMD          string           `json:"body_md"`
	Revision        int              `json:"revision"`
	PermissionLevel domain.Privilege `json:"permission_level"`
	CreatedOn       time.Time        `json:"created_on"`
	UpdatedOn       time.Time        `json:"updated_on"`
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

func NewPage(slug string, body string) Page {
	now := time.Now()

	return Page{
		Slug:            slug,
		BodyMD:          body,
		Revision:        0,
		PermissionLevel: domain.PGuest,
		CreatedOn:       now,
		UpdatedOn:       now,
	}
}
