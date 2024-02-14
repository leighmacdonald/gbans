package asset

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type assetUsecase struct {
	ar domain.AssetRepository
}

func NewAssetUsecase(assetRepository domain.AssetRepository) domain.AssetUsecase {
	return &assetUsecase{ar: assetRepository}
}

func (s assetUsecase) GetAsset(ctx context.Context, uuid uuid.UUID) (domain.Asset, io.Reader, error) {
	asset, errAsset := s.ar.GetAsset(ctx, uuid)
	if errAsset != nil {
		return asset, nil, errAsset
	}

	content, err := s.ar.Get(ctx, asset.Bucket, asset.Name)
	if err != nil {
		return asset, nil, err
	}

	return asset, content, nil
}

func (s assetUsecase) SaveAsset(ctx context.Context, bucket string, asset *domain.Asset, content []byte) error {
	if errPut := s.ar.Put(ctx, bucket, asset.Name, bytes.NewReader(content), asset.Size, asset.MimeType); errPut != nil {
		return errPut
	}

	if errSaveAsset := s.ar.SaveAsset(ctx, asset); errSaveAsset != nil {
		return errSaveAsset
	}

	return nil
}

func (s assetUsecase) DropAsset(ctx context.Context, asset *domain.Asset) error {
	if err := s.ar.Remove(ctx, asset.Bucket, asset.Name); err != nil {
		return err
	}

	if err := s.ar.DeleteAsset(ctx, asset); err != nil {
		return err
	}

	return nil
}

func GenerateFileMeta(body io.Reader, name string) (string, string, int64, error) {
	content, errRead := io.ReadAll(body)
	if errRead != nil {
		return "", "", 0, errors.Join(errRead, domain.ErrReadContent)
	}

	mime := mimetype.Detect(content)

	if !strings.HasSuffix(strings.ToLower(name), mime.Extension()) {
		name += mime.Extension()
	}

	return name, mime.String(), int64(len(content)), nil
}
