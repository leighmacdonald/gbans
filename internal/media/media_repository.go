package media

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type mediaRepository struct {
	db database.Database
}

func NewMediaRepository(db database.Database) domain.MediaRepository {
	return mediaRepository{db: db}
}

func (r mediaRepository) SaveMedia(ctx context.Context, media *domain.Media) error {
	if media.MediaID > 0 {
		return r.db.DBErr(r.db.ExecUpdateBuilder(ctx, r.db.
			Builder().
			Update("media").
			Set("author_id", media.AuthorID).
			Set("mime_type", media.MimeType).
			Set("name", media.Name).
			Set("contents", media.Contents).
			Set("size", media.Size).
			Set("deleted", media.Deleted).
			Set("updated_on", media.UpdatedOn).
			Set("asset_id", media.Asset.AssetID).
			Where(sq.Eq{"media_id": media.MediaID})))
	}

	const query = `
		INSERT INTO media (
		    author_id, mime_type, name, contents, size, deleted, created_on, updated_on, asset_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING media_id
	`

	return r.db.DBErr(r.db.QueryRow(ctx, query,
		media.AuthorID,
		media.MimeType,
		media.Name,
		media.Contents,
		media.Size,
		media.Deleted,
		media.CreatedOn,
		media.UpdatedOn,
		media.Asset.AssetID,
	).
		Scan(&media.MediaID))
}

func (r mediaRepository) GetMediaByName(ctx context.Context, name string, media *domain.Media) error {
	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("media_id", "author_id", "name", "size", "mime_type", "contents", "deleted",
			"created_on", "updated_on").
		From("media").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"name": name}}))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	var authorID int64

	if errScan := row.Scan(
		&media.MediaID,
		&authorID,
		&media.Name,
		&media.Size,
		&media.MimeType,
		&media.Contents,
		&media.Deleted,
		&media.CreatedOn,
		&media.UpdatedOn,
	); errScan != nil {
		return r.db.DBErr(errScan)
	}

	media.AuthorID = steamid.New(authorID)

	return nil
}

func (r mediaRepository) GetMediaByID(ctx context.Context, mediaID int, media *domain.Media) error {
	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("m.media_id", "m.author_id", "m.name", "m.size", "m.mime_type", "m.contents",
			"m.deleted", "m.created_on", "m.updated_on", "a.name", "a.size", "a.mime_type",
			"a.path", "a.bucket", "a.old_id").
		From("media m").
		LeftJoin("asset a USING(asset_id)").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"m.media_id": mediaID}}))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	var authorID int64

	if errScan := row.Scan(
		&media.MediaID,
		&authorID,
		&media.Name,
		&media.Size,
		&media.MimeType,
		&media.Contents,
		&media.Deleted,
		&media.CreatedOn,
		&media.UpdatedOn,
		&media.Asset.Name,
		&media.Asset.Size,
		&media.Asset.MimeType,
		&media.Asset.Path,
		&media.Asset.Bucket,
		&media.Asset.OldID,
	); errScan != nil {
		return r.db.DBErr(errScan)
	}

	media.AuthorID = steamid.New(authorID)

	return nil
}

func (r mediaRepository) GetMediaByAssetID(ctx context.Context, uuid uuid.UUID, media *domain.Media) error {
	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("m.media_id", "m.author_id", "m.name", "m.size", "m.mime_type", "m.contents",
			"m.deleted", "m.created_on", "m.updated_on", "a.name", "a.size", "a.mime_type", "a.path",
			"a.bucket", "a.old_id").
		From("media m").
		LeftJoin("asset a USING(asset_id)").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"m.asset_id": uuid}}))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	media.Asset = domain.Asset{AssetID: uuid}

	var authorID int64

	if errScan := row.Scan(
		&media.MediaID,
		&authorID,
		&media.Name,
		&media.Size,
		&media.MimeType,
		&media.Contents,
		&media.Deleted,
		&media.CreatedOn,
		&media.UpdatedOn,
		&media.Asset.Name,
		&media.Asset.Size,
		&media.Asset.MimeType,
		&media.Asset.Path,
		&media.Asset.Bucket,
		&media.Asset.OldID,
	); errScan != nil {
		return r.db.DBErr(errScan)
	}

	media.AuthorID = steamid.New(authorID)

	return nil
}
