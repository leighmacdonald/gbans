package wiki

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type wikiUsecase struct {
	repository WikiRepository
}

func NewWikiUsecase(repository WikiRepository) WikiUsecase {
	return &wikiUsecase{repository: repository}
}

func (w *wikiUsecase) GetWikiPageBySlug(ctx context.Context, user domain.PersonInfo, slug string) (Page, error) {
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

func (w *wikiUsecase) DeleteWikiPageBySlug(ctx context.Context, slug string) error {
	return w.repository.DeleteWikiPageBySlug(ctx, slug)
}

func (w *wikiUsecase) SaveWikiPage(ctx context.Context, user domain.PersonInfo, slug string, body string, level domain.Privilege) (Page, error) {
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
