package media

import (
	"context"
	"errors"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"golang.org/x/exp/slices"
)

type mediaUsecase struct {
	mr     domain.MediaRepository
	au     domain.AssetUsecase
	bucket string
}

func NewMediaUsecase(bucket string, mr domain.MediaRepository, au domain.AssetUsecase) domain.MediaUsecase {
	return &mediaUsecase{
		mr:     mr,
		au:     au,
		bucket: bucket,
	}
}

func (u mediaUsecase) Create(ctx context.Context, steamId steamid.SID64, name string, mimeType string, content []byte,
	mimeTypesAllowed []string,
) (*domain.Media, error) {
	if len(mimeTypesAllowed) > 0 && !slices.Contains(mimeTypesAllowed, strings.ToLower(mimeType)) {
		return nil, domain.ErrMimeTypeNotAllowed
	}

	media, errMedia := domain.NewMedia(steamId, name, mimeType, content)
	if errMedia != nil {
		return nil, errMedia
	}

	asset, errAsset := domain.NewAsset(media.Contents, u.bucket, "")
	if errAsset != nil {
		return nil, errors.Join(errAsset, domain.ErrAssetCreateFailed)
	}

	if errSave := u.au.SaveAsset(ctx, u.bucket, &asset, content); errSave != nil {
		return nil, errSave
	}

	media.Asset = asset

	media.Contents = nil

	if errSave := u.mr.SaveMedia(ctx, &media); errSave != nil {
		return nil, errSave
	}

	return &media, nil
}

func (u mediaUsecase) GetMediaByAssetID(ctx context.Context, uuid uuid.UUID, media *domain.Media) error {
	return u.mr.GetMediaByAssetID(ctx, uuid, media)
}

func (u mediaUsecase) GetMediaByName(ctx context.Context, name string, media *domain.Media) error {
	return u.mr.GetMediaByName(ctx, name, media)
}

func (u mediaUsecase) GetMediaByID(ctx context.Context, mediaID int, media *domain.Media) error {
	return u.mr.GetMediaByID(ctx, mediaID, media)
}
