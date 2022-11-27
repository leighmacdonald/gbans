package store

import (
	"context"
	"github.com/pkg/errors"
)

func (database *pgStore) SetPatreonAuth(ctx context.Context, accessToken string, refreshToken string) error {
	query, args, errQuery := sb.
		Update("patreon_auth").
		Set("creator_access_token", accessToken).
		Set("creator_refresh_token", refreshToken).ToSql()
	if errQuery != nil {
		return errors.Wrap(errQuery, "Failed to create patreon auth update query")
	}
	return Err(database.Exec(ctx, query, args...))
}

func (database *pgStore) GetPatreonAuth(ctx context.Context) (string, string, error) {
	query, args, errQuery := sb.
		Select("creator_access_token", "creator_refresh_token").From("patreon_auth").ToSql()
	if errQuery != nil {
		return "", "", errors.Wrap(errQuery, "Failed to create patreon auth select query")
	}
	var (
		creatorAccessToken  string
		creatorRefreshToken string
	)
	if errScan := database.QueryRow(ctx, query, args...).Scan(&creatorAccessToken, &creatorRefreshToken); errScan != nil {
		return "", "", errors.Wrap(errQuery, "Failed to query patreon auth")
	}
	return creatorAccessToken, creatorRefreshToken, nil
}
