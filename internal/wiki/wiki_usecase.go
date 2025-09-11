package wiki

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/person/permission"
)

type WikiUsecase struct {
	repository wikiRepository
}

func NewWikiUsecase(repository wikiRepository) *WikiUsecase {
	return &WikiUsecase{repository: repository}
}

func (w *WikiUsecase) GetWikiPageBySlug(ctx context.Context, user person.PersonInfo, slug string) (Page, error) {
	slug = strings.ToLower(slug)
	if slug[0] == '/' {
		slug = slug[1:]
	}

	page, errGetWikiSlug := w.repository.GetWikiPageBySlug(ctx, slug)
	if errGetWikiSlug != nil {
		return page, errGetWikiSlug
	}

	if !user.HasPermission(page.PermissionLevel) {
		return page, domain.ErrPermissionDenied
	}

	return page, nil
}

func (w *WikiUsecase) DeleteWikiPageBySlug(ctx context.Context, slug string) error {
	return w.repository.DeleteWikiPageBySlug(ctx, slug)
}

func (w *WikiUsecase) SaveWikiPage(ctx context.Context, user person.PersonInfo, slug string, body string, level permission.Privilege) (Page, error) {
	if slug == "" || body == "" {
		return Page{}, domain.ErrInvalidParameter
	}

	page, errGetWikiSlug := w.GetWikiPageBySlug(ctx, user, slug)
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
