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
	assetRepository domain.AssetRepository
}

func NewAssetUsecase(assetRepository domain.AssetRepository) domain.AssetUsecase {
	return &assets{assetRepository: assetRepository}
}

func (s assets) Create(ctx context.Context, author steamid.SteamID, bucket domain.Bucket, fileName string, content io.ReadSeeker) (domain.Asset, error) {
	if bucket != "demos" && bucket != "media" {
		return domain.Asset{}, domain.ErrBucketType
	}

	if fileName == "" {
		return domain.Asset{}, domain.ErrAssetName
	}

	if bucket != "demos" && !author.Valid() {
		// Non demo assets must have a real author
		return domain.Asset{}, domain.ErrInvalidAuthorSID
	}

	asset, errAsset := NewAsset(author, fileName, bucket, content)
	if errAsset != nil {
		return domain.Asset{}, errAsset
	}

	newAsset, errPut := s.assetRepository.Put(ctx, asset, content)
	if errPut != nil {
		return domain.Asset{}, errPut
	}

	slog.Debug("Created new asset",
		slog.String("name", asset.Name), slog.String("asset_id", asset.AssetID.String()))

	return newAsset, nil
}

func (s assets) Get(ctx context.Context, uuid uuid.UUID) (domain.Asset, io.ReadSeeker, error) {
	if uuid.IsNil() {
		return domain.Asset{}, nil, domain.ErrUUIDInvalid
	}

	asset, reader, errAsset := s.assetRepository.Get(ctx, uuid)
	if errAsset != nil {
		return asset, nil, errAsset
	}

	return asset, reader, nil
}

func (s assets) Delete(ctx context.Context, assetID uuid.UUID) (int64, error) {
	if assetID.IsNil() {
		return 0, domain.ErrUUIDInvalid
	}

	size, err := s.assetRepository.Delete(ctx, assetID)
	if err != nil {
		return 0, err
	}

	slog.Debug("Removed demo asset", slog.String("asset_id", assetID.String()), slog.String("size", humanize.Bytes(uint64(size))))

	return size, nil
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

func NewAsset(author steamid.SteamID, name string, bucket domain.Bucket, contentReader io.ReadSeeker) (domain.Asset, error) {
	mType, errMime := mimetype.DetectReader(contentReader)
	if errMime != nil {
		return domain.Asset{}, errors.Join(errMime, domain.ErrMimeTypeReadFailed)
	}

	_, _ = contentReader.Seek(0, 0)

	size, errSize := io.Copy(io.Discard, contentReader)
	if errSize != nil {
		return domain.Asset{}, errors.Join(errSize, domain.ErrCopyFileContent)
	}

	if bucket == domain.BucketMedia && size > maxMediaFileSize || bucket == domain.BucketDemo && size > maxDemoFileSize {
		return domain.Asset{}, domain.ErrAssetTooLarge
	}

	_, _ = contentReader.Seek(0, 0)

	hash, errHash := generateFileHash(contentReader)
	if errHash != nil {
		return domain.Asset{}, errHash
	}

	curTime := time.Now()

	newID, errID := uuid.NewV4()
	if errID != nil {
		return domain.Asset{}, errors.Join(errID, domain.ErrUUIDCreate)
	}

	if name == domain.UnknownMediaTag {
		name = fmt.Sprintf("%x%s", hash, mType.Extension())
	}

	asset := domain.Asset{
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
