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
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct {
	db       database.Database
	rootPath string
}

func NewLocalRepository(database database.Database, rootPath string) Repository {
	return Repository{db: database, rootPath: rootPath}
}

func (l *Repository) Put(ctx context.Context, asset Asset, body io.ReadSeeker) (Asset, error) {
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
		return Asset{}, errors.Join(errFile, ErrCreateAddFile)
	}

	defer func() {
		if errClose := file.Close(); errClose != nil {
			slog.Error("failed to close asset file", log.ErrAttr(errClose))
		}
	}()

	_, _ = body.Seek(0, 0)

	_, errWrite := io.Copy(file, body)
	if errWrite != nil {
		return Asset{}, errors.Join(errWrite, ErrCopyFileContent)
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

func (l Repository) Delete(ctx context.Context, assetID uuid.UUID) (int64, error) {
	asset, errAsset := l.getAssetByUUID(ctx, assetID)
	if errAsset != nil {
		return 0, errAsset
	}

	query := l.db.Builder().Delete("asset").Where(sq.Eq{"asset_id": assetID})

	if errExec := l.db.ExecDeleteBuilder(ctx, query); errExec != nil {
		return 0, database.DBErr(errExec)
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

		return 0, errors.Join(errRemove, ErrDeleteAssetFile)
	}

	return asset.Size, nil
}

func (l Repository) Init(_ context.Context) error {
	if l.rootPath == "" {
		return ErrPathInvalid
	}

	if errDir := os.MkdirAll(l.rootPath, 0o770); errDir != nil {
		return errors.Join(errDir, fmt.Errorf("%w: %s", ErrCreateAssetPath, l.rootPath))
	}

	return nil
}

func (l Repository) Get(ctx context.Context, assetID uuid.UUID) (Asset, io.ReadSeeker, error) {
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
		return Asset{}, nil, errors.Join(errReader, ErrOpenFile)
	}

	return asset, reader, nil
}

func (l Repository) GenAssetPath(hash string) (string, error) {
	if len(hash) < 2 {
		return "", httphelper.ErrInvalidParameter
	}

	firstLevel := hash[0:2]
	secondLevel := hash[2:4]
	fullPath := path.Join(l.rootPath, firstLevel, secondLevel)

	if err := os.MkdirAll(fullPath, 0o770); err != nil {
		return "", errors.Join(err, ErrCreateAssetPath)
	}

	return path.Join(fullPath, hash), nil
}

func (l Repository) getAssetByUUID(ctx context.Context, assetID uuid.UUID) (Asset, error) {
	query, args, errSQL := l.db.Builder().
		Select("asset_id", "bucket", "author_id", "mime_type", "name", "size", "hash", "created_on", "updated_on", "is_private").
		From("asset").
		Where(sq.Eq{"asset_id": assetID}).
		ToSql()
	if errSQL != nil {
		return Asset{}, database.DBErr(errSQL)
	}

	var (
		asset    Asset
		authorID int64
	)

	if errScan := l.db.QueryRow(ctx, query, args...).
		Scan(&asset.AssetID, &asset.Bucket, &authorID, &asset.MimeType, &asset.Name,
			&asset.Size, &asset.Hash, &asset.CreatedOn, &asset.UpdatedOn, &asset.IsPrivate); errScan != nil {
		return Asset{}, database.DBErr(errScan)
	}

	asset.AuthorID = steamid.New(authorID)

	assetPath, errAssetPath := l.GenAssetPath(asset.HashString())
	if errAssetPath != nil {
		return Asset{}, errAssetPath
	}

	asset.LocalPath = assetPath

	return asset, nil
}

func (l Repository) getAssetByHash(ctx context.Context, hash []byte) (Asset, error) {
	query, args, errSQL := l.db.Builder().
		Select("asset_id", "bucket", "author_id", "mime_type", "name", "size", "hash", "is_private", "created_on", "updated_on").
		From("asset").
		Where(sq.Eq{"hash": hash}).
		ToSql()
	if errSQL != nil {
		return Asset{}, database.DBErr(errSQL)
	}

	var (
		asset    Asset
		authorID int64
	)

	if errScan := l.db.QueryRow(ctx, query, args...).
		Scan(&asset.AssetID, &asset.Bucket, &authorID, &asset.MimeType, &asset.Name,
			&asset.Size, &asset.Hash, &asset.IsPrivate, &asset.CreatedOn, &asset.UpdatedOn); errScan != nil {
		return Asset{}, database.DBErr(errScan)
	}

	asset.AuthorID = steamid.New(authorID)

	assetPath, errAssetPath := l.GenAssetPath(asset.HashString())
	if errAssetPath != nil {
		return Asset{}, errAssetPath
	}

	asset.LocalPath = assetPath

	return asset, nil
}

func (l Repository) saveAssetToDB(ctx context.Context, asset Asset) error {
	query := l.db.Builder().Insert("asset").SetMap(map[string]any{
		"asset_id":   asset.AssetID,
		"hash":       asset.Hash,
		"author_id":  asset.AuthorID.Int64(),
		"bucket":     asset.Bucket,
		"mime_type":  asset.MimeType,
		"is_private": asset.IsPrivate,
		"name":       asset.Name,
		"size":       asset.Size,
		"created_on": asset.CreatedOn,
		"updated_on": asset.UpdatedOn,
	})

	if errInsert := l.db.ExecInsertBuilder(ctx, query); errInsert != nil {
		return database.DBErr(errInsert)
	}

	return nil
}
