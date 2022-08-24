package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	log "github.com/sirupsen/logrus"
	"strings"
)

func (database *pgStore) GetWikiPageBySlug(ctx context.Context, slug string, page *wiki.Page) error {
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
	if errQuery := database.conn.QueryRow(ctx, query, args...).Scan(&page.Slug, &page.BodyMD, &page.Revision,
		&page.CreatedOn, &page.UpdatedOn); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) DeleteWikiPageBySlug(ctx context.Context, slug string) error {
	query, args, errQueryArgs := sb.
		Delete("wiki").
		Where(sq.Eq{"slug": slug}).
		ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	if _, errExec := database.conn.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	log.Debugf("Wiki slug deleted: %s", slug)
	return nil
}

func (database *pgStore) SaveWikiPage(ctx context.Context, page *wiki.Page) error {
	query, args, errQueryArgs := sb.
		Insert("wiki").
		Columns("slug", "body_md", "revision", "created_on", "updated_on").
		Values(page.Slug, page.BodyMD, page.Revision, page.CreatedOn, page.UpdatedOn).
		ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	_, errQueryRow := database.conn.Exec(ctx, query, args...)
	if errQueryRow != nil {
		return Err(errQueryRow)
	}
	log.Debugf("Wiki page saved: %s", util.SanitizeLog(page.Slug))
	return nil
}

func (database *pgStore) SaveMedia(ctx context.Context, media *model.Media) error {
	const query = `
		INSERT INTO media (
		    author_id, mime_type, name, contents, size, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING media_id
	`
	if errQuery := database.conn.QueryRow(ctx, query,
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
	log.WithFields(log.Fields{
		"wiki_media_id": media.MediaId,
		"author_id":     media.AuthorId,
		"name":          util.SanitizeLog(media.Name),
		"size":          media.Size,
		"mime":          util.SanitizeLog(media.MimeType),
	}).Infof("Wiki media created")
	return nil
}

func (database *pgStore) GetMediaByName(ctx context.Context, name string, media *model.Media) error {
	const query = `
		SELECT 
		   media_id, author_id, name, size, mime_type, contents, deleted, created_on, updated_on
		FROM media
		WHERE deleted = false AND name = $1`
	return Err(database.conn.QueryRow(ctx, query, name).Scan(
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
