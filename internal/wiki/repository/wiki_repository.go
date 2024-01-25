package repository

import (
	"context"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type wikiRepository struct {
	store.Database
}

func NewWikiRepository(database store.Database) domain.WikiRepository {
	return &wikiRepository{Database: database}
}

func (s *wikiRepository) GetWikiPageBySlug(ctx context.Context, slug string, page *wiki.Page) error {
	row, errQuery := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("slug", "body_md", "revision", "created_on", "updated_on", "permission_level").
		From("wiki").
		Where(sq.Eq{"lower(slug)": strings.ToLower(slug)}).
		OrderBy("revision desc").
		Limit(1))
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return errs.DBErr(row.Scan(&page.Slug, &page.BodyMD, &page.Revision, &page.CreatedOn, &page.UpdatedOn, &page.PermissionLevel))
}

func (s *wikiRepository) DeleteWikiPageBySlug(ctx context.Context, slug string) error {
	if errExec := s.ExecDeleteBuilder(ctx, s.
		Builder().
		Delete("wiki").
		Where(sq.Eq{"slug": slug})); errExec != nil {
		return errs.DBErr(errExec)
	}

	return nil
}

func (s *wikiRepository) SaveWikiPage(ctx context.Context, page *wiki.Page) error {
	errQueryRow := s.ExecInsertBuilder(ctx, s.
		Builder().
		Insert("wiki").
		Columns("slug", "body_md", "revision", "created_on", "updated_on", "permission_level").
		Values(page.Slug, page.BodyMD, page.Revision, page.CreatedOn, page.UpdatedOn, page.PermissionLevel))
	if errQueryRow != nil {
		return errs.DBErr(errQueryRow)
	}

	return nil
}

func (s *wikiRepository) SaveMedia(ctx context.Context, media *domain.Media) error {
	if media.MediaID > 0 {
		return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
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

	return errs.DBErr(s.QueryRow(ctx, query,
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

func (s *wikiRepository) GetMediaByAssetID(ctx context.Context, uuid uuid.UUID, media *domain.Media) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("m.media_id", "m.author_id", "m.name", "m.size", "m.mime_type", "m.contents",
			"m.deleted", "m.created_on", "m.updated_on", "a.name", "a.size", "a.mime_type", "a.path",
			"a.bucket", "a.old_id").
		From("media m").
		LeftJoin("asset a USING(asset_id)").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"m.asset_id": uuid}}))
	if errRow != nil {
		return errs.DBErr(errRow)
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
		return errs.DBErr(errScan)
	}

	media.AuthorID = steamid.New(authorID)

	return nil
}

func (s *wikiRepository) GetMediaByName(ctx context.Context, name string, media *domain.Media) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("media_id", "author_id", "name", "size", "mime_type", "contents", "deleted",
			"created_on", "updated_on").
		From("media").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"name": name}}))
	if errRow != nil {
		return errs.DBErr(errRow)
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
		return errs.DBErr(errScan)
	}

	media.AuthorID = steamid.New(authorID)

	return nil
}

func (s *wikiRepository) GetMediaByID(ctx context.Context, mediaID int, media *domain.Media) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("m.media_id", "m.author_id", "m.name", "m.size", "m.mime_type", "m.contents",
			"m.deleted", "m.created_on", "m.updated_on", "a.name", "a.size", "a.mime_type",
			"a.path", "a.bucket", "a.old_id").
		From("media m").
		LeftJoin("asset a USING(asset_id)").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"m.media_id": mediaID}}))
	if errRow != nil {
		return errs.DBErr(errRow)
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
		return errs.DBErr(errScan)
	}

	media.AuthorID = steamid.New(authorID)

	return nil
}
