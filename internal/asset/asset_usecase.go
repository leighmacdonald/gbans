package asset

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type assets struct {
	repository AssetRepository
}

func NewAssetUsecase(assetRepository AssetRepository) AssetUsecase {
	return &assets{repository: assetRepository}
}

func (s assets) Create(ctx context.Context, author steamid.SteamID, bucket Bucket, fileName string, content io.ReadSeeker) (Asset, error) {
	if bucket != "demos" && bucket != "media" {
		return Asset{}, domain.ErrBucketType
	}

	if fileName == "" {
		return Asset{}, domain.ErrAssetName
	}

	if bucket != "demos" && !author.Valid() {
		// Non demo assets must have a real author
		return Asset{}, domain.ErrInvalidAuthorSID
	}

	asset, errAsset := NewAsset(author, fileName, bucket, content)
	if errAsset != nil {
		return Asset{}, errAsset
	}

	newAsset, errPut := s.repository.Put(ctx, asset, content)
	if errPut != nil {
		return Asset{}, errPut
	}

	slog.Debug("Created new asset",
		slog.String("name", asset.Name), slog.String("asset_id", asset.AssetID.String()))

	return newAsset, nil
}

func (s assets) Get(ctx context.Context, uuid uuid.UUID) (Asset, io.ReadSeeker, error) {
	if uuid.IsNil() {
		return Asset{}, nil, domain.ErrUUIDInvalid
	}

	asset, reader, errAsset := s.repository.Get(ctx, uuid)
	if errAsset != nil {
		return asset, nil, errAsset
	}

	return asset, reader, nil
}

func (s assets) Delete(ctx context.Context, assetID uuid.UUID) (int64, error) {
	if assetID.IsNil() {
		return 0, domain.ErrUUIDInvalid
	}

	size, err := s.repository.Delete(ctx, assetID)
	if err != nil {
		return 0, err
	}

	slog.Debug("Removed demo asset", slog.String("asset_id", assetID.String()), slog.String("size", humanize.Bytes(uint64(size)))) //nolint:gosec

	return size, nil
}

func (s assets) GenAssetPath(hash string) (string, error) {
	return s.repository.GenAssetPath(hash)
}

func generateFileHash(file io.Reader) ([]byte, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, domain.ErrHashFileContent
	}

	return hasher.Sum(nil), nil
}

const (
	maxMediaFileSize = 25000000
	maxDemoFileSize  = 500000000
)

func NewAsset(author steamid.SteamID, name string, bucket Bucket, contentReader io.ReadSeeker) (Asset, error) {
	mType, errMime := mimetype.DetectReader(contentReader)
	if errMime != nil {
		return Asset{}, errors.Join(errMime, domain.ErrMimeTypeReadFailed)
	}

	_, _ = contentReader.Seek(0, 0)

	size, errSize := io.Copy(io.Discard, contentReader)
	if errSize != nil {
		return Asset{}, errors.Join(errSize, domain.ErrCopyFileContent)
	}

	if bucket == BucketMedia && size > maxMediaFileSize || bucket == BucketDemo && size > maxDemoFileSize {
		return Asset{}, domain.ErrAssetTooLarge
	}

	_, _ = contentReader.Seek(0, 0)

	hash, errHash := generateFileHash(contentReader)
	if errHash != nil {
		return Asset{}, errHash
	}

	curTime := time.Now()

	newID, errID := uuid.NewV4()
	if errID != nil {
		return Asset{}, errors.Join(errID, domain.ErrUUIDCreate)
	}

	if name == UnknownMediaTag {
		name = fmt.Sprintf("%x%s", hash, mType.Extension())
	}

	asset := Asset{
		AssetID:   newID,
		Bucket:    bucket,
		AuthorID:  author,
		Hash:      hash,
		IsPrivate: false,
		MimeType:  mType.String(),
		Name:      strings.ReplaceAll(name, " ", "_"),
		Size:      size,
		CreatedOn: curTime,
		UpdatedOn: curTime,
	}

	return asset, nil
}
