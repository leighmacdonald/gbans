package patreon

import (
	"context"
	"errors"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type patreonRepository struct {
	db database.Database
}

func NewPatreonRepository(database database.Database) domain.PatreonRepository {
	return &patreonRepository{db: database}
}

func (r patreonRepository) SetPatreonAuth(ctx context.Context, accessToken string, refreshToken string) error {
	return r.db.DBErr(r.db.ExecUpdateBuilder(ctx, r.db.
		Builder().
		Update("patreon_auth").
		Set("creator_access_token", accessToken).
		Set("creator_refresh_token", refreshToken)))
}

func (r patreonRepository) GetPatreonAuth(ctx context.Context) (string, string, error) {
	query, args, errQuery := r.db.
		Builder().
		Select("creator_access_token", "creator_refresh_token").From("patreon_auth").ToSql()
	if errQuery != nil {
		return "", "", errors.Join(errQuery, domain.ErrCreateQuery)
	}

	var (
		creatorAccessToken  string
		creatorRefreshToken string
	)

	if errScan := r.db.
		QueryRow(ctx, query, args...).
		Scan(&creatorAccessToken, &creatorRefreshToken); errScan != nil {
		return "", "", errors.Join(errQuery, domain.ErrQueryPatreon)
	}

	return creatorAccessToken, creatorRefreshToken, nil
}
