package usecase

import (
	"bytes"
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type AssetUsecase struct {
	ar domain.AssetRepository
}

func (s AssetUsecase) GetAsset(ctx context.Context, uuid uuid.UUID) (*domain.Asset, error) {
	panic("DropAsset")
}

func NewAssetUsecase(ar domain.AssetRepository) domain.AssetUsecase {
	return &AssetUsecase{ar: ar}
}

func (s AssetUsecase) SaveAsset(ctx context.Context, bucket string, asset *domain.Asset, content []byte) error {
	if errPut := s.ar.Put(ctx, bucket, asset.Name, bytes.NewReader(content), asset.Size, asset.MimeType); errPut != nil {
		return errPut
	}

	if errSaveAsset := s.ar.SaveAsset(ctx, asset); errSaveAsset != nil {
		return errSaveAsset
	}

	return nil
}

func (s AssetUsecase) DropAsset(ctx context.Context, asset *domain.Asset) error {
	if err := s.ar.Remove(ctx, asset.Bucket, asset.Name); err != nil {
		return err
	}

	if err := s.ar.DeleteAsset(ctx, asset); err != nil {
		return err
	}

	return nil
}
