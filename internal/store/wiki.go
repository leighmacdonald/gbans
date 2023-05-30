package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/gabriel-vasile/mimetype"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"strings"
	"time"
)

const unknownMediaTag = "__unknown__"

var MediaSafeMimeTypesImages = []string{
	"image/gif",
	"image/jpeg",
	"image/png",
	"image/webp",
}

func NewMedia(author steamid.SID64, name string, mime string, content []byte) (Media, error) {
	mtype := mimetype.Detect(content)
	if !mtype.Is(mime) && mime != unknownMediaTag {
		// Should never actually happen unless user is trying nefarious stuff.
		return Media{}, errors.New("Detected mimetype different than provided")
	}
	t0 := config.Now()
	return Media{
		AuthorId:  author,
		MimeType:  mtype.String(),
		Name:      strings.Replace(name, " ", "_", -1),
		Size:      int64(len(content)),
		Contents:  content,
		Deleted:   false,
		CreatedOn: t0,
		UpdatedOn: t0,
	}, nil
}

type Media struct {
	MediaId   int           `json:"media_id"`
	AuthorId  steamid.SID64 `json:"author_id,string"`
	MimeType  string        `json:"mime_type"`
	Contents  []byte        `json:"-"`
	Name      string        `json:"name"`
	Size      int64         `json:"size"`
	Deleted   bool          `json:"deleted"`
	CreatedOn time.Time     `json:"created_on"`
	UpdatedOn time.Time     `json:"updated_on"`
}

func GetWikiPageBySlug(ctx context.Context, slug string, page *wiki.Page) error {
	query, args, errQueryArgs := sb.
		Select("slug", "body_md", "revision", "created_on", "updated_on").
		From("wiki").
		Where(sq.Eq{"lower(slug)": strings.ToLower(slug)}).
		OrderBy("revision desc").
		Limit(1).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errQuery := QueryRow(ctx, query, args...).Scan(&page.Slug, &page.BodyMD, &page.Revision,
		&page.CreatedOn, &page.UpdatedOn); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func DeleteWikiPageBySlug(ctx context.Context, slug string) error {
	query, args, errQueryArgs := sb.
		Delete("wiki").
		Where(sq.Eq{"slug": slug}).
		ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	if errExec := Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	logger.Info("Wiki slug deleted", zap.String("slug", slug))
	return nil
}

func SaveWikiPage(ctx context.Context, page *wiki.Page) error {
	query, args, errQueryArgs := sb.
		Insert("wiki").
		Columns("slug", "body_md", "revision", "created_on", "updated_on").
		Values(page.Slug, page.BodyMD, page.Revision, page.CreatedOn, page.UpdatedOn).
		ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	errQueryRow := Exec(ctx, query, args...)
	if errQueryRow != nil {
		return Err(errQueryRow)
	}
	logger.Info("Wiki page saved", zap.String("slug", util.SanitizeLog(page.Slug)))
	return nil
}

func SaveMedia(ctx context.Context, media *Media) error {
	const query = `
		INSERT INTO media (
		    author_id, mime_type, name, contents, size, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING media_id
	`
	if errQuery := QueryRow(ctx, query,
		media.AuthorId,
		media.MimeType,
		media.Name,
		media.Contents,
		media.Size,
		media.Deleted,
		media.CreatedOn,
		media.UpdatedOn,
	).Scan(&media.MediaId); errQuery != nil {
		return Err(errQuery)
	}
	logger.Info("Wiki media created",
		zap.Int("wiki_media_id", media.MediaId),
		zap.Int64("author_id", media.AuthorId.Int64()),
		zap.String("name", util.SanitizeLog(media.Name)),
		zap.Int64("size", media.Size),
		zap.String("mime", util.SanitizeLog(media.MimeType)),
	)
	return nil
}

func GetMediaByName(ctx context.Context, name string, media *Media) error {
	const query = `
		SELECT 
		   media_id, author_id, name, size, mime_type, contents, deleted, created_on, updated_on
		FROM media
		WHERE deleted = false AND name = $1`
	return Err(QueryRow(ctx, query, name).Scan(
		&media.MediaId,
		&media.AuthorId,
		&media.Name,
		&media.Size,
		&media.MimeType,
		&media.Contents,
		&media.Deleted,
		&media.CreatedOn,
		&media.UpdatedOn,
	))
}

func GetMediaById(ctx context.Context, mediaId int, media *Media) error {
	const query = `
		SELECT 
		   media_id, author_id, name, size, mime_type, contents, deleted, created_on, updated_on
		FROM media
		WHERE deleted = false AND media_id = $1`
	return Err(QueryRow(ctx, query, mediaId).Scan(
		&media.MediaId,
		&media.AuthorId,
		&media.Name,
		&media.Size,
		&media.MimeType,
		&media.Contents,
		&media.Deleted,
		&media.CreatedOn,
		&media.UpdatedOn,
	))
}
