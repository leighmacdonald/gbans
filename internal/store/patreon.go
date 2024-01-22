package store

import (
	"context"
	"errors"

	"github.com/leighmacdonald/gbans/internal/errs"
)

func (s Stores) SetPatreonAuth(ctx context.Context, accessToken string, refreshToken string) error {
	return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
		Builder().
		Update("patreon_auth").
		Set("creator_access_token", accessToken).
		Set("creator_refresh_token", refreshToken)))
}

func (s Stores) GetPatreonAuth(ctx context.Context) (string, string, error) {
	query, args, errQuery := s.
		Builder().
		Select("creator_access_token", "creator_refresh_token").From("patreon_auth").ToSql()
	if errQuery != nil {
		return "", "", errors.Join(errQuery, errors.New("Failed to create patreon auth select query"))
	}

	var (
		creatorAccessToken  string
		creatorRefreshToken string
	)

	if errScan := s.
		QueryRow(ctx, query, args...).
		Scan(&creatorAccessToken, &creatorRefreshToken); errScan != nil {
		return "", "", errors.Join(errQuery, errors.New("Failed to query patreon auth"))
	}

	return creatorAccessToken, creatorRefreshToken, nil
}
