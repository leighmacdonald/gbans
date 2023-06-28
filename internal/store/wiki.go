package store

import (
	"context"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gabriel-vasile/mimetype"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const unknownMediaTag = "__unknown__"

var MediaSafeMimeTypesImages = []string{
	"image/gif",
	"image/jpeg",
	"image/png",
	"image/webp",
}

func NewMedia(author steamid.SID64, name string, mime string, content []byte) (Media, error) {
	mType := mimetype.Detect(content)
	if !mType.Is(mime) && mime != unknownMediaTag {
		// Should never actually happen unless user is trying nefarious stuff.
		return Media{}, errors.New("Detected mimetype different than provided")
	}
	t0 := config.Now()
	return Media{
		AuthorID:  author,
		MimeType:  mType.String(),
		Name:      strings.ReplaceAll(name, " ", "_"),
		Size:      int64(len(content)),
		Contents:  content,
		Deleted:   false,
		CreatedOn: t0,
		UpdatedOn: t0,
	}, nil
}

type Media struct {
	MediaID   int           `json:"media_id"`
	AuthorID  steamid.SID64 `json:"author_id,string"`
	MimeType  string        `json:"mime_type"`
	Contents  []byte        `json:"-"`
	Name      string        `json:"name"`
	Size      int64         `json:"size"`
	Deleted   bool          `json:"deleted"`
	CreatedOn time.Time     `json:"created_on"`
	UpdatedOn time.Time     `json:"updated_on"`
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
		return errQueryArgs
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
		return errQueryArgs
	}
	errQueryRow := db.Exec(ctx, query, args...)
	if errQueryRow != nil {
		return Err(errQueryRow)
	}
	db.log.Info("Wiki page saved", zap.String("slug", util.SanitizeLog(page.Slug)))
	return nil
}

func (db *Store) SaveMedia(ctx context.Context, media *Media) error {
	const query = `
		INSERT INTO media (
		    author_id, mime_type, name, contents, size, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
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
	).Scan(&media.MediaID); errQuery != nil {
		return Err(errQuery)
	}
	db.log.Info("Wiki media created",
		zap.Int("wiki_media_id", media.MediaID),
		zap.Int64("author_id", media.AuthorID.Int64()),
		zap.String("name", util.SanitizeLog(media.Name)),
		zap.Int64("size", media.Size),
		zap.String("mime", util.SanitizeLog(media.MimeType)),
	)
	return nil
}

func (db *Store) GetMediaByName(ctx context.Context, name string, media *Media) error {
	const query = `
		SELECT 
		   media_id, author_id, name, size, mime_type, contents, deleted, created_on, updated_on
		FROM media
		WHERE deleted = false AND name = $1`
	return Err(db.QueryRow(ctx, query, name).Scan(
		&media.MediaID,
		&media.AuthorID,
		&media.Name,
		&media.Size,
		&media.MimeType,
		&media.Contents,
		&media.Deleted,
		&media.CreatedOn,
		&media.UpdatedOn,
	))
}

func (db *Store) GetMediaByID(ctx context.Context, mediaID int, media *Media) error {
	const query = `
		SELECT 
		   media_id, author_id, name, size, mime_type, contents, deleted, created_on, updated_on
		FROM media
		WHERE deleted = false AND media_id = $1`
	return Err(db.QueryRow(ctx, query, mediaID).Scan(
		&media.MediaID,
		&media.AuthorID,
		&media.Name,
		&media.Size,
		&media.MimeType,
		&media.Contents,
		&media.Deleted,
		&media.CreatedOn,
		&media.UpdatedOn,
	))
}
