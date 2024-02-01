package domain

import (
	"context"
	"time"
)

const unknownMediaTag = "__unknown__"

type WikiRepository interface {
	GetWikiPageBySlug(ctx context.Context, slug string) (WikiPage, error)
	DeleteWikiPageBySlug(ctx context.Context, slug string) error
	SaveWikiPage(ctx context.Context, page *WikiPage) error
}

type WikiUsecase interface {
	GetWikiPageBySlug(ctx context.Context, slug string) (WikiPage, error)
	DeleteWikiPageBySlug(ctx context.Context, slug string) error
	SaveWikiPage(ctx context.Context, page *WikiPage) error
}

// RootSlug is the top-most (index) page of the wiki.
const RootSlug = "home"

type WikiPage struct {
	Slug            string    `json:"slug"`
	BodyMD          string    `json:"body_md"`
	Revision        int       `json:"revision"`
	PermissionLevel Privilege `json:"permission_level"`
	CreatedOn       time.Time `json:"created_on"`
	UpdatedOn       time.Time `json:"updated_on"`
}

func (page *WikiPage) NewRevision() WikiPage {
	return WikiPage{
		Slug:      page.Slug,
		BodyMD:    page.BodyMD,
		Revision:  page.Revision + 1,
		CreatedOn: page.CreatedOn,
		UpdatedOn: time.Now(),
	}
}

func NewWikiPage(slug string, body string) WikiPage {
	now := time.Now()

	return WikiPage{
		Slug:            slug,
		BodyMD:          body,
		Revision:        0,
		PermissionLevel: PGuest,
		CreatedOn:       now,
		UpdatedOn:       now,
	}
}
