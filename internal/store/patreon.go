package store

import (
	"context"

	"github.com/pkg/errors"
)

func SetPatreonAuth(ctx context.Context, database Store, accessToken string, refreshToken string) error {
	return DBErr(database.ExecUpdateBuilder(ctx, database.
		Builder().
		Update("patreon_auth").
		Set("creator_access_token", accessToken).
		Set("creator_refresh_token", refreshToken)))
}

func GetPatreonAuth(ctx context.Context, database Store) (string, string, error) {
	query, args, errQuery := database.
		Builder().
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
