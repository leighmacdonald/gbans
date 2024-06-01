package patreon

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type patreonRepository struct {
	db database.Database
}

func NewPatreonRepository(database database.Database) domain.PatreonRepository {
	return &patreonRepository{db: database}
}

func (r patreonRepository) OldAuths(ctx context.Context) ([]domain.PatreonCredential, error) {
	const query = `SELECT steam_id, patreon_id, access_token,  refresh_token,
                          expires_in, scope, token_type, version, created_on, updated_on FROM auth_patreon
					WHERE to_timestamp(extract(epoch from updated_on) + expires_in) < (now() + interval '7 days');`

	rows, errRows := r.db.Query(ctx, query)
	if errRows != nil {
		return nil, r.db.DBErr(errRows)
	}

	var credentials []domain.PatreonCredential

	for rows.Next() {
		var creds domain.PatreonCredential
		if errScan := rows.Scan(&creds.SteamID, &creds.PatreonID, &creds.AccessToken, &creds.RefreshToken, &creds.ExpiresIn,
			&creds.Scope, &creds.TokenType, &creds.Version, &creds.CreatedOn, &creds.UpdatedOn); errScan != nil {
			return credentials, r.db.DBErr(errScan)
		}

		credentials = append(credentials, creds)
	}

	return credentials, nil
}

func (r patreonRepository) DeleteTokens(ctx context.Context, steamID steamid.SteamID) error {
	query, vars, errQuery := r.db.Builder().
		Delete("auth_patreon").
		Where(sq.Eq{"steam_id": steamID}).
		ToSql()
	if errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	return r.db.Exec(ctx, query, vars...)
}

func (r patreonRepository) SetPatreonAuth(ctx context.Context, accessToken string, refreshToken string) error {
	return r.db.DBErr(r.db.ExecUpdateBuilder(ctx, r.db.
		Builder().
		Update("auth_patreon").
		Set("creator_access_token", accessToken).
		Set("creator_refresh_token", refreshToken)))
}

func (r patreonRepository) GetPatreonAuth(ctx context.Context) (string, string, error) {
	query, args, errQuery := r.db.
		Builder().
		Select("creator_access_token", "creator_refresh_token").
		From("auth_patreon").
		ToSql()
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

func (r patreonRepository) SaveTokens(ctx context.Context, creds domain.PatreonCredential) error {
	const query = `INSERT INTO auth_patreon (
                          steam_id, patreon_id, access_token, refresh_token,
                          expires_in, scope, token_type, version, created_on, updated_on)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT (steam_id)
				DO UPDATE SET patreon_id = $2, access_token= $3, refresh_token = $4,
				              expires_in = $5, scope = $6, token_type = $7, version = $8, updated_on = $10
				`

	return r.db.DBErr(r.db.Exec(ctx, query, creds.SteamID.Int64(), creds.PatreonID, creds.AccessToken, creds.RefreshToken,
		creds.ExpiresIn, creds.Scope, creds.TokenType, creds.Version, creds.CreatedOn, creds.UpdatedOn,
	))
}

func (r patreonRepository) GetTokens(ctx context.Context, steamID steamid.SteamID) (domain.PatreonCredential, error) {
	row, errRow := r.db.QueryRowBuilder(ctx, r.db.Builder().
		Select("patreon_id", "access_token", "refresh_token",
			"expires_in", "scope", "token_type", "version", "created_on", "updated_on").
		From("auth_patreon").
		Where(sq.Eq{"steam_id": steamID.Int64()}))
	if errRow != nil {
		return domain.PatreonCredential{}, r.db.DBErr(errRow)
	}

	var creds domain.PatreonCredential

	creds.SteamID = steamID

	if err := row.Scan(&creds.PatreonID, &creds.AccessToken, &creds.RefreshToken, &creds.ExpiresIn, &creds.Scope, &creds.TokenType,
		&creds.Version, &creds.CreatedOn, &creds.UpdatedOn); err != nil {
		return domain.PatreonCredential{}, r.db.DBErr(errRow)
	}

	return creds, nil
}
