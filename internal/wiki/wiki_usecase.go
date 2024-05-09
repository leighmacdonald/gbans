package wiki

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type wikiUsecase struct {
	wikiRepo domain.WikiRepository
}

func NewWikiUsecase(repository domain.WikiRepository) domain.WikiUsecase {
	return &wikiUsecase{wikiRepo: repository}
}

func (w *wikiUsecase) GetWikiPageBySlug(ctx context.Context, user domain.PersonInfo, slug string) (domain.WikiPage, error) {
	slug = strings.ToLower(slug)
	if slug[0] == '/' {
		slug = slug[1:]
	}

	page, errGetWikiSlug := w.wikiRepo.GetWikiPageBySlug(ctx, slug)
	if errGetWikiSlug != nil {
		return page, errGetWikiSlug
	}

	if !user.HasPermission(page.PermissionLevel) {
		return page, domain.ErrPermissionDenied
	}

	return page, nil
}

func (w *wikiUsecase) DeleteWikiPageBySlug(ctx context.Context, slug string) error {
	return w.wikiRepo.DeleteWikiPageBySlug(ctx, slug)
}

func (w *wikiUsecase) SaveWikiPage(ctx context.Context, user domain.PersonInfo, slug string, body string, level domain.Privilege) (domain.WikiPage, error) {
	if slug == "" || body == "" {
		return domain.WikiPage{}, domain.ErrInvalidParameter
	}

	page, errGetWikiSlug := w.GetWikiPageBySlug(ctx, user, slug)
	if errGetWikiSlug != nil {
		if errors.Is(errGetWikiSlug, domain.ErrNoResult) {
			page.CreatedOn = time.Now()
			page.Revision += 1
			page.Slug = slug
		} else {
			return page, domain.ErrInternal
		}
	} else {
		page = page.NewRevision()
	}

	page.PermissionLevel = level
	page.BodyMD = body

	if errSave := w.wikiRepo.SaveWikiPage(ctx, &page); errSave != nil {
		return page, errSave
	}

	return page, nil
}
