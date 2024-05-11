package wiki

import (
	"context"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type wikiRepository struct {
	db database.Database
}

func NewWikiRepository(database database.Database) domain.WikiRepository {
	return &wikiRepository{db: database}
}

func (r *wikiRepository) GetWikiPageBySlug(ctx context.Context, slug string) (domain.WikiPage, error) {
	var page domain.WikiPage

	row, errQuery := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("slug", "body_md", "revision", "created_on", "updated_on", "permission_level").
		From("wiki").
		Where(sq.Eq{"lower(slug)": strings.ToLower(slug)}).
		OrderBy("revision desc").
		Limit(1))
	if errQuery != nil {
		return page, r.db.DBErr(errQuery)
	}

	if err := row.Scan(&page.Slug, &page.BodyMD, &page.Revision, &page.CreatedOn, &page.UpdatedOn, &page.PermissionLevel); err != nil {
		return page, r.db.DBErr(err)
	}

	return page, nil
}

func (r *wikiRepository) DeleteWikiPageBySlug(ctx context.Context, slug string) error {
	if errExec := r.db.ExecDeleteBuilder(ctx, r.db.
		Builder().
		Delete("wiki").
		Where(sq.Eq{"slug": slug})); errExec != nil {
		return r.db.DBErr(errExec)
	}

	return nil
}

func (r *wikiRepository) SaveWikiPage(ctx context.Context, page *domain.WikiPage) error {
	errQueryRow := r.db.ExecInsertBuilder(ctx, r.db.
		Builder().
		Insert("wiki").
		Columns("slug", "body_md", "revision", "created_on", "updated_on", "permission_level").
		Values(page.Slug, page.BodyMD, page.Revision, page.CreatedOn, page.UpdatedOn, page.PermissionLevel))
	if errQueryRow != nil {
		return r.db.DBErr(errQueryRow)
	}

	return nil
}
