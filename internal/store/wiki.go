package store

import (
	"context"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const unknownMediaTag = "__unknown__"

func NewMedia(author steamid.SID64, name string, mime string, content []byte) (Media, error) {
	mType := mimetype.Detect(content)
	if !mType.Is(mime) && mime != unknownMediaTag {
		// Should never actually happen unless user is trying nefarious stuff.
		return Media{}, errors.New("Detected mimetype different than provided")
	}

	curTime := time.Now()

	return Media{
		AuthorID:  author,
		MimeType:  mType.String(),
		Name:      strings.ReplaceAll(name, " ", "_"),
		Size:      int64(len(content)),
		Contents:  content,
		Deleted:   false,
		CreatedOn: curTime,
		UpdatedOn: curTime,
		Asset:     Asset{},
	}, nil
}

type Media struct {
	MediaID   int           `json:"media_id"`
	AuthorID  steamid.SID64 `json:"author_id"`
	MimeType  string        `json:"mime_type"`
	Contents  []byte        `json:"-"`
	Name      string        `json:"name"`
	Size      int64         `json:"size"`
	Deleted   bool          `json:"deleted"`
	CreatedOn time.Time     `json:"created_on"`
	UpdatedOn time.Time     `json:"updated_on"`
	Asset     Asset         `json:"asset"`
}

func (db *Store) GetWikiPageBySlug(ctx context.Context, slug string, page *wiki.Page) error {
	query, args, errQueryArgs := db.sb.
		Select("slug", "body_md", "revision", "created_on", "updated_on").
		From("wiki").
		Where(sq.Eq{"lower(slug)": strings.ToLower(slug)}).
		OrderBy("revision desc").
		Limit(1).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}

	if errQuery := db.QueryRow(ctx, query, args...).Scan(&page.Slug, &page.BodyMD, &page.Revision,
		&page.CreatedOn, &page.UpdatedOn); errQuery != nil {
		return Err(errQuery)
	}

	return nil
}

func (db *Store) DeleteWikiPageBySlug(ctx context.Context, slug string) error {
	query, args, errQueryArgs := db.sb.
		Delete("wiki").
		Where(sq.Eq{"slug": slug}).
		ToSql()

	if errQueryArgs != nil {
		return errors.Wrap(errQueryArgs, "Failed to generate query")
	}

	if errExec := db.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}

	db.log.Info("Wiki slug deleted", zap.String("slug", slug))

	return nil
}

func (db *Store) SaveWikiPage(ctx context.Context, page *wiki.Page) error {
	query, args, errQueryArgs := db.sb.
		Insert("wiki").
		Columns("slug", "body_md", "revision", "created_on", "updated_on").
		Values(page.Slug, page.BodyMD, page.Revision, page.CreatedOn, page.UpdatedOn).
		ToSql()
	if errQueryArgs != nil {
		return errors.Wrap(errQueryArgs, "Failed to generate query")
	}

	errQueryRow := db.Exec(ctx, query, args...)
	if errQueryRow != nil {
		return Err(errQueryRow)
	}

	db.log.Info("Wiki page saved", zap.String("slug", util.SanitizeLog(page.Slug)))

	return nil
}

func (db *Store) SaveMedia(ctx context.Context, media *Media) error {
	if media.MediaID > 0 {
		const query = `
			UPDATE media 
			SET author_id = $2, mime_type = $3, name = $4, contents = $5, size = $6, deleted = $7, updated_on = $8, asset_id = $9
			WHERE media_id = $1`

		if errQuery := db.Exec(ctx, query, media.MediaID, media.AuthorID, media.MimeType, media.Name,
			media.Contents, media.Size, media.Deleted, media.UpdatedOn, media.Asset.AssetID); errQuery != nil {
			return errQuery
		}
	} else {
		const query = `
		INSERT INTO media (
		    author_id, mime_type, name, contents, size, deleted, created_on, updated_on, asset_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING media_id
	`

		if errQuery := db.QueryRow(ctx, query,
			media.AuthorID,
			media.MimeType,
			media.Name,
			media.Contents,
			media.Size,
			media.Deleted,
			media.CreatedOn,
			media.UpdatedOn,
			media.Asset.AssetID,
		).Scan(&media.MediaID); errQuery != nil {
			return Err(errQuery)
		}

		db.log.Info("Media created",
			zap.Int("media_id", media.MediaID),
			zap.Int64("author_id", media.AuthorID.Int64()),
			zap.String("name", util.SanitizeLog(media.Name)),
			zap.Int64("size", media.Size),
			zap.String("mime", util.SanitizeLog(media.MimeType)),
		)
	}

	return nil
}

func (db *Store) GetMediaByAssetID(ctx context.Context, uuid uuid.UUID, media *Media) error {
	const query = `
		SELECT 
		   m.media_id, m.author_id, m.name, m.size, m.mime_type, m.contents, m.deleted, m.created_on, m.updated_on,
		   a.name, a.size, a.mime_type, a.path, a.bucket, a.old_id
		FROM media m
		LEFT JOIN asset a on a.asset_id = m.asset_id
		WHERE deleted = false AND m.asset_id = $1`

	media.Asset = Asset{AssetID: uuid}

	var authorID int64

	if errRow := db.
		QueryRow(ctx, query, uuid).Scan(
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
	); errRow != nil {
		return Err(errRow)
	}

	media.AuthorID = steamid.New(authorID)

	return nil
}

func (db *Store) GetMediaByName(ctx context.Context, name string, media *Media) error {
	const query = `
		SELECT 
		   media_id, author_id, name, size, mime_type, contents, deleted, created_on, updated_on
		FROM media
		WHERE deleted = false AND name = $1`

	var authorID int64

	if errRow := db.QueryRow(ctx, query, name).Scan(
		&media.MediaID,
		&authorID,
		&media.Name,
		&media.Size,
		&media.MimeType,
		&media.Contents,
		&media.Deleted,
		&media.CreatedOn,
		&media.UpdatedOn,
	); errRow != nil {
		return Err(errRow)
	}

	media.AuthorID = steamid.New(authorID)

	return nil
}

func (db *Store) GetMediaByID(ctx context.Context, mediaID int, media *Media) error {
	const query = `
		SELECT 
		   m.media_id, m.author_id, m.name, m.size, m.mime_type, m.contents, m.deleted, m.created_on, m.updated_on,
		    a.name, a.size, a.mime_type, a.path, a.bucket, a.old_id
		FROM media m
		LEFT JOIN asset a on a.asset_id = m.asset_id
		WHERE deleted = false AND m.media_id = $1`

	var authorID int64

	if errRow := db.QueryRow(ctx, query, mediaID).Scan(
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
	); errRow != nil {
		return Err(errRow)
	}

	media.AuthorID = steamid.New(authorID)

	return nil
}
