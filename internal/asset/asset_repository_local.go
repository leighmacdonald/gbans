package asset

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type localRepository struct {
	db database.Database
	cu domain.ConfigUsecase
}

func NewLocalRepository(database database.Database, configUsecase domain.ConfigUsecase) domain.AssetRepository {
	return &localRepository{db: database, cu: configUsecase}
}

func (l localRepository) Put(ctx context.Context, asset domain.Asset, body io.ReadSeeker) (domain.Asset, error) {
	existing, errExisting := l.getAssetByHash(ctx, asset.Hash)
	if errExisting == nil {
		return existing, nil
	}

	if errExisting != nil && !errors.Is(errExisting, domain.ErrNoResult) {
		return domain.Asset{}, errExisting
	}

	outPath, errOutPath := l.genAssetPath(asset.HashString())
	if errOutPath != nil {
		return domain.Asset{}, errOutPath
	}

	file, errFile := os.Create(outPath)
	if errFile != nil {
		return domain.Asset{}, errors.Join(errFile, domain.ErrCreateAddFile)
	}

	defer func() {
		if errClose := file.Close(); errClose != nil {
			slog.Error("failed to close asset file", log.ErrAttr(errClose))
		}
	}()

	_, _ = body.Seek(0, 0)

	_, errWrite := io.Copy(file, body)
	if errWrite != nil {
		return domain.Asset{}, errors.Join(errWrite, domain.ErrCopyFileContent)
	}

	if errSave := l.saveAssetToDB(ctx, asset); errSave != nil {
		if errRemove := os.Remove(outPath); errRemove != nil {
			return domain.Asset{}, errors.Join(errRemove, errSave)
		}

		return domain.Asset{}, errSave
	}

	asset.LocalPath = outPath

	return asset, nil
}

func (l localRepository) Delete(ctx context.Context, assetID uuid.UUID) error {
	asset, errAsset := l.getAssetByUUID(ctx, assetID)
	if errAsset != nil {
		return errAsset
	}

	query := l.db.Builder().Delete("asset").Where(sq.Eq{"asset_id": assetID})

	if errExec := l.db.ExecDeleteBuilder(ctx, query); errExec != nil {
		return l.db.DBErr(errExec)
	}

	assetPath, errAssetPath := l.genAssetPath(asset.HashString())
	if errAssetPath != nil {
		return errAssetPath
	}

	if errRemove := os.Remove(assetPath); errRemove != nil {
		return errors.Join(errRemove, domain.ErrDeleteAssetFile)
	}

	return nil
}

func (l localRepository) Init(_ context.Context) error {
	rootPath := l.cu.Config().LocalStore.PathRoot
	if rootPath == "" {
		return domain.ErrPathInvalid
	}

	if errDir := os.MkdirAll(rootPath, 0o770); errDir != nil {
		return errors.Join(errDir, fmt.Errorf("%w: %s", domain.ErrCreateAssetPath, rootPath))
	}

	return nil
}

func (l localRepository) Get(ctx context.Context, assetID uuid.UUID) (domain.Asset, io.ReadSeeker, error) {
	asset, errAsset := l.getAssetByUUID(ctx, assetID)
	if errAsset != nil {
		return domain.Asset{}, nil, errAsset
	}

	assetPath, errAssetPath := l.genAssetPath(asset.HashString())
	if errAssetPath != nil {
		return domain.Asset{}, nil, errAssetPath
	}

	reader, errReader := os.Open(assetPath)
	if errReader != nil {
		return domain.Asset{}, nil, errors.Join(errReader, domain.ErrOpenFile)
	}

	return asset, reader, nil
}

func (l localRepository) genAssetPath(hash string) (string, error) {
	if len(hash) < 2 {
		return "", domain.ErrInvalidParameter
	}

	firstLevel := hash[0:2]
	secondLevel := hash[2:4]
	root := l.cu.Config().LocalStore.PathRoot

	fullPath := path.Join(root, firstLevel, secondLevel)

	if err := os.MkdirAll(fullPath, 0o770); err != nil {
		return "", errors.Join(err, domain.ErrCreateAssetPath)
	}

	return path.Join(fullPath, hash), nil
}

func (l localRepository) getAssetByUUID(ctx context.Context, assetID uuid.UUID) (domain.Asset, error) {
	query, args, errSQL := l.db.Builder().
		Select("asset_id", "bucket", "author_id", "mime_type", "name", "size", "hash", "created_on", "updated_on").
		From("asset").
		Where(sq.Eq{"asset_id": assetID}).
		ToSql()
	if errSQL != nil {
		return domain.Asset{}, l.db.DBErr(errSQL)
	}

	var (
		asset    domain.Asset
		authorID int64
	)

	if errScan := l.db.QueryRow(ctx, query, args...).
		Scan(&asset.AssetID, &asset.Bucket, &authorID, &asset.MimeType, &asset.Name,
			&asset.Size, &asset.Hash, &asset.CreatedOn, &asset.UpdatedOn); errScan != nil {
		return domain.Asset{}, l.db.DBErr(errScan)
	}

	asset.AuthorID = steamid.New(authorID)

	assetPath, errAssetPath := l.genAssetPath(asset.HashString())
	if errAssetPath != nil {
		return domain.Asset{}, errAssetPath
	}

	asset.LocalPath = assetPath

	return asset, nil
}

func (l localRepository) getAssetByHash(ctx context.Context, hash []byte) (domain.Asset, error) {
	query, args, errSQL := l.db.Builder().
		Select("asset_id", "bucket", "author_id", "mime_type", "name", "size", "hash", "is_private", "created_on", "updated_on").
		From("asset").
		Where(sq.Eq{"hash": hash}).
		ToSql()
	if errSQL != nil {
		return domain.Asset{}, l.db.DBErr(errSQL)
	}

	var (
		asset    domain.Asset
		authorID int64
	)

	if errScan := l.db.QueryRow(ctx, query, args...).
		Scan(&asset.AssetID, &asset.Bucket, &authorID, &asset.MimeType, &asset.Name,
			&asset.Size, &asset.Hash, &asset.IsPrivate, &asset.CreatedOn, &asset.UpdatedOn); errScan != nil {
		return domain.Asset{}, l.db.DBErr(errScan)
	}

	asset.AuthorID = steamid.New(authorID)

	assetPath, errAssetPath := l.genAssetPath(asset.HashString())
	if errAssetPath != nil {
		return domain.Asset{}, errAssetPath
	}

	asset.LocalPath = assetPath

	return asset, nil
}

func (l localRepository) saveAssetToDB(ctx context.Context, asset domain.Asset) error {
	query := l.db.Builder().Insert("asset").SetMap(map[string]interface{}{
		"asset_id":   asset.AssetID,
		"hash":       asset.Hash,
		"author_id":  asset.AuthorID.Int64(),
		"bucket":     asset.Bucket,
		"mime_type":  asset.MimeType,
		"name":       asset.Name,
		"size":       asset.Size,
		"created_on": asset.CreatedOn,
		"updated_on": asset.UpdatedOn,
	})

	if errInsert := l.db.ExecInsertBuilder(ctx, query); errInsert != nil {
		return l.db.DBErr(errInsert)
	}

	return nil
}
