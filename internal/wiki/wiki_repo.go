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

func (r *Repository) Page(ctx context.Context, slug string) (Page, error) {
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

func (r *Repository) Delete(ctx context.Context, slug string) error {
	if errExec := r.db.ExecDeleteBuilder(ctx, r.db.Builder().
		Delete("wiki").
		Where(sq.Eq{"lower(slug)": strings.ToLower(slug)})); errExec != nil {
		return database.DBErr(errExec)
	}

	return nil
}

func (r *Repository) Save(ctx context.Context, page *Page) error {
	const query = `
		INSERT INTO wiki (slug, body_md, revision, created_on, updated_on, permission_level)
		VALUES ($1, $2, $3, $4, $5, $6)`
	if errQueryRow := r.db.Exec(ctx, query, page.Slug, page.BodyMD, page.Revision, page.CreatedOn, page.UpdatedOn, page.PermissionLevel); errQueryRow != nil {
		return database.DBErr(errQueryRow)
	}

	return nil
}
