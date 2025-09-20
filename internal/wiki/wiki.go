package wiki

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/microcosm-cc/bluemonday"
)

func Render(page Page) []byte {
	unsafeHTML := markdown.ToHTML([]byte(page.BodyMD), NewParser(), nil)

	return bluemonday.UGCPolicy().SanitizeBytes(unsafeHTML)
}

func NewParser() *parser.Parser {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.Tables

	return parser.NewWithExtensions(extensions)
}

// RootSlug is the top-most (index) page of the wiki.
const RootSlug = "home"

type Page struct {
	Slug            string               `json:"slug"`
	BodyMD          string               `json:"body_md"`
	Revision        int                  `json:"revision"`
	PermissionLevel permission.Privilege `json:"permission_level"`
	CreatedOn       time.Time            `json:"created_on"`
	UpdatedOn       time.Time            `json:"updated_on"`
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
		PermissionLevel: permission.PGuest,
		CreatedOn:       now,
		UpdatedOn:       now,
	}
}

type Wiki struct {
	repository Repository
}

func NewWiki(repository Repository) Wiki {
	return Wiki{repository: repository}
}

func (w *Wiki) BySlug(ctx context.Context, user domain.PersonInfo, slug string) (Page, error) {
	slug = strings.ToLower(slug)
	if slug[0] == '/' {
		slug = slug[1:]
	}

	page, errGetWikiSlug := w.repository.GetWikiPageBySlug(ctx, slug)
	if errGetWikiSlug != nil {
		return page, errGetWikiSlug
	}

	if !user.HasPermission(page.PermissionLevel) {
		return page, permission.ErrPermissionDenied
	}

	return page, nil
}

func (w *Wiki) DeleteBySlug(ctx context.Context, slug string) error {
	return w.repository.DeleteWikiPageBySlug(ctx, slug)
}

func (w *Wiki) Save(ctx context.Context, user domain.PersonInfo, slug string, body string, level permission.Privilege) (Page, error) {
	if slug == "" || body == "" {
		return Page{}, domain.ErrInvalidParameter
	}

	page, errGetWikiSlug := w.BySlug(ctx, user, slug)
	if errGetWikiSlug != nil {
		if errors.Is(errGetWikiSlug, database.ErrNoResult) {
			page.CreatedOn = time.Now()
			page.Revision++
			page.Slug = slug
		} else {
			return page, httphelper.ErrInternal // TODO better error
		}
	} else {
		page = page.NewRevision()
	}

	page.PermissionLevel = level
	page.BodyMD = body

	if errSave := w.repository.SaveWikiPage(ctx, &page); errSave != nil {
		return page, errSave
	}

	return page, nil
}
