package wiki

import (
	"context"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
)

type Repository struct {
	db database.Database
}

func NewRepository(database database.Database) Repository {
	return Repository{db: database}
}

func (r *Repository) GetWikiPageBySlug(ctx context.Context, slug string) (Page, error) {
	var page Page

	row, errQuery := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("slug", "body_md", "revision", "created_on", "updated_on", "permission_level").
		From("wiki").
		Where(sq.Eq{"lower(slug)": strings.ToLower(slug)}).
		OrderBy("revision desc").
		Limit(1))
	if errQuery != nil {
		return page, database.DBErr(errQuery)
	}

	if err := row.Scan(&page.Slug, &page.BodyMD, &page.Revision, &page.CreatedOn, &page.UpdatedOn, &page.PermissionLevel); err != nil {
		return page, database.DBErr(err)
	}

	return page, nil
}

func (r *Repository) DeleteWikiPageBySlug(ctx context.Context, slug string) error {
	if errExec := r.db.ExecDeleteBuilder(ctx, r.db.
		Builder().
		Delete("wiki").
		Where(sq.Eq{"slug": slug})); errExec != nil {
		return database.DBErr(errExec)
	}

	return nil
}

func (r *Repository) SaveWikiPage(ctx context.Context, page *Page) error {
	errQueryRow := r.db.ExecInsertBuilder(ctx, r.db.
		Builder().
		Insert("wiki").
		Columns("slug", "body_md", "revision", "created_on", "updated_on", "permission_level").
		Values(page.Slug, page.BodyMD, page.Revision, page.CreatedOn, page.UpdatedOn, page.PermissionLevel))
	if errQueryRow != nil {
		return database.DBErr(errQueryRow)
	}

	return nil
}
