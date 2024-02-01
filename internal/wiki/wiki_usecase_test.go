package wiki_test

import (
	"context"
	"testing"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/domain/mocks"
	"github.com/leighmacdonald/gbans/internal/wiki"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetWikiPageBySlug(t *testing.T) {
	mockRepo := new(mocks.MockWikiRepository)
	mockPage := domain.NewWikiPage("test", util.SecureRandomString(100))

	t.Run("success", func(t *testing.T) {
		mockRepo.
			On("GetWikiPageBySlug", mock.Anything, mock.AnythingOfType("string")).
			Return(mockPage, nil).
			Once()

		u := wiki.NewWikiUsecase(mockRepo)
		page, errPage := u.GetWikiPageBySlug(context.TODO(), mockPage.Slug)
		require.NoError(t, errPage)
		require.EqualValues(t, page, mockPage)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.
			On("GetWikiPageBySlug", mock.Anything, mock.AnythingOfType("string")).
			Return(domain.WikiPage{}, domain.ErrNoResult).
			Once()

		_, errPage := wiki.NewWikiUsecase(mockRepo).GetWikiPageBySlug(context.TODO(), "invalid_slug")
		require.Error(t, errPage)
	})
}
