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
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type localRepository struct {
	db database.Database
	cu *config.ConfigUsecase
}

func NewLocalRepository(database database.Database, configUsecase *config.ConfigUsecase) AssetRepository {
	return &localRepository{db: database, cu: configUsecase}
}

func (l *localRepository) Put(ctx context.Context, asset Asset, body io.ReadSeeker) (Asset, error) {
	existing, errExisting := l.getAssetByHash(ctx, asset.Hash)
	if errExisting == nil {
		return existing, nil
	}

	if !errors.Is(errExisting, database.ErrNoResult) {
		return Asset{}, errExisting
	}

	outPath, errOutPath := l.GenAssetPath(asset.HashString())
	if errOutPath != nil {
		return Asset{}, errOutPath
	}

	file, errFile := os.Create(outPath)
	if errFile != nil {
		return Asset{}, errors.Join(errFile, domain.ErrCreateAddFile)
	}

	defer func() {
		if errClose := file.Close(); errClose != nil {
			slog.Error("failed to close asset file", log.ErrAttr(errClose))
		}
	}()

	_, _ = body.Seek(0, 0)

	_, errWrite := io.Copy(file, body)
	if errWrite != nil {
		return Asset{}, errors.Join(errWrite, domain.ErrCopyFileContent)
	}

	if errSave := l.saveAssetToDB(ctx, asset); errSave != nil {
		if errRemove := os.Remove(outPath); errRemove != nil {
			return Asset{}, errors.Join(errRemove, errSave)
		}

		return Asset{}, errSave
	}

	asset.LocalPath = outPath

	return asset, nil
}

func (l localRepository) Delete(ctx context.Context, assetID uuid.UUID) (int64, error) {
	asset, errAsset := l.getAssetByUUID(ctx, assetID)
	if errAsset != nil {
		return 0, errAsset
	}

	query := l.db.Builder().Delete("asset").Where(sq.Eq{"asset_id": assetID})

	if errExec := l.db.ExecDeleteBuilder(ctx, nil, query); errExec != nil {
		return 0, l.db.DBErr(errExec)
	}

	assetPath, errAssetPath := l.GenAssetPath(asset.HashString())
	if errAssetPath != nil {
		return 0, errAssetPath
	}

	if errRemove := os.Remove(assetPath); errRemove != nil {
		var e *os.PathError
		if errors.As(errRemove, &e) && errors.Is(e.Err, os.ErrNotExist) {
			return 0, nil
		}

		return 0, errors.Join(errRemove, domain.ErrDeleteAssetFile)
	}

	return asset.Size, nil
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

func (l localRepository) Get(ctx context.Context, assetID uuid.UUID) (Asset, io.ReadSeeker, error) {
	asset, errAsset := l.getAssetByUUID(ctx, assetID)
	if errAsset != nil {
		return Asset{}, nil, errAsset
	}

	assetPath, errAssetPath := l.GenAssetPath(asset.HashString())
	if errAssetPath != nil {
		return Asset{}, nil, errAssetPath
	}

	reader, errReader := os.Open(assetPath)
	if errReader != nil {
		return Asset{}, nil, errors.Join(errReader, domain.ErrOpenFile)
	}

	return asset, reader, nil
}

func (l localRepository) GenAssetPath(hash string) (string, error) {
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

func (l localRepository) getAssetByUUID(ctx context.Context, assetID uuid.UUID) (Asset, error) {
	query, args, errSQL := l.db.Builder().
		Select("asset_id", "bucket", "author_id", "mime_type", "name", "size", "hash", "created_on", "updated_on").
		From("asset").
		Where(sq.Eq{"asset_id": assetID}).
		ToSql()
	if errSQL != nil {
		return Asset{}, l.db.DBErr(errSQL)
	}

	var (
		asset    Asset
		authorID int64
	)

	if errScan := l.db.QueryRow(ctx, nil, query, args...).
		Scan(&asset.AssetID, &asset.Bucket, &authorID, &asset.MimeType, &asset.Name,
			&asset.Size, &asset.Hash, &asset.CreatedOn, &asset.UpdatedOn); errScan != nil {
		return Asset{}, l.db.DBErr(errScan)
	}

	asset.AuthorID = steamid.New(authorID)

	assetPath, errAssetPath := l.GenAssetPath(asset.HashString())
	if errAssetPath != nil {
		return Asset{}, errAssetPath
	}

	asset.LocalPath = assetPath

	return asset, nil
}

func (l localRepository) getAssetByHash(ctx context.Context, hash []byte) (Asset, error) {
	query, args, errSQL := l.db.Builder().
		Select("asset_id", "bucket", "author_id", "mime_type", "name", "size", "hash", "is_private", "created_on", "updated_on").
		From("asset").
		Where(sq.Eq{"hash": hash}).
		ToSql()
	if errSQL != nil {
		return Asset{}, l.db.DBErr(errSQL)
	}

	var (
		asset    Asset
		authorID int64
	)

	if errScan := l.db.QueryRow(ctx, nil, query, args...).
		Scan(&asset.AssetID, &asset.Bucket, &authorID, &asset.MimeType, &asset.Name,
			&asset.Size, &asset.Hash, &asset.IsPrivate, &asset.CreatedOn, &asset.UpdatedOn); errScan != nil {
		return Asset{}, l.db.DBErr(errScan)
	}

	asset.AuthorID = steamid.New(authorID)

	assetPath, errAssetPath := l.GenAssetPath(asset.HashString())
	if errAssetPath != nil {
		return Asset{}, errAssetPath
	}

	asset.LocalPath = assetPath

	return asset, nil
}

func (l localRepository) saveAssetToDB(ctx context.Context, asset Asset) error {
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

	if errInsert := l.db.ExecInsertBuilder(ctx, nil, query); errInsert != nil {
		return l.db.DBErr(errInsert)
	}

	return nil
}
