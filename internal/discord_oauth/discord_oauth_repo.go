package discordoauth

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type DiscordOAuthRepository struct {
	db database.Database
}

func NewRepository(database database.Database) DiscordOAuthRepository {
	return DiscordOAuthRepository{db: database}
}

func (d DiscordOAuthRepository) SaveUserDetail(ctx context.Context, detail DiscordUserDetail) error {
	const query = `INSERT INTO discord_user (
                          steam_id, discord_id, username, avatar,
                          publicflags, mfa_enabled, premium_type, created_on, updated_on)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				ON CONFLICT (discord_id)
				DO UPDATE SET discord_id = $2, username= $3, avatar = $4,
				              publicflags = $5, mfa_enabled = $6, premium_type = $7, updated_on = $9
				`

	return database.DBErr(d.db.Exec(ctx, nil, query, detail.SteamID.Int64(), detail.ID, detail.Username, detail.Avatar,
		detail.PublicFlags, detail.MfaEnabled, detail.PremiumType, detail.CreatedOn, detail.UpdatedOn,
	))
}

func (d DiscordOAuthRepository) GetUserDetail(ctx context.Context, steamID steamid.SteamID) (DiscordUserDetail, error) {
	row, errRow := d.db.QueryRowBuilder(ctx, nil, d.db.Builder().
		Select("discord_id", "username", "avatar",
			"publicflags", "mfa_enabled", "premium_type", "created_on", "updated_on").
		From("discord_user").
		Where(sq.Eq{"steam_id": steamID.Int64()}))
	if errRow != nil {
		return DiscordUserDetail{}, database.DBErr(errRow)
	}

	var detail DiscordUserDetail

	detail.SteamID = steamID

	if err := row.Scan(&detail.ID, &detail.Username, &detail.Avatar, &detail.PublicFlags, &detail.MfaEnabled, &detail.PremiumType,
		&detail.CreatedOn, &detail.UpdatedOn); err != nil {
		return DiscordUserDetail{}, database.DBErr(errRow)
	}

	return detail, nil
}

func (d DiscordOAuthRepository) DeleteUserDetail(ctx context.Context, steamID steamid.SteamID) error {
	return database.DBErr(d.db.ExecDeleteBuilder(ctx, nil, d.db.Builder().
		Delete("discord_user").
		Where(sq.Eq{"steam_id": steamID})))
}

func (d DiscordOAuthRepository) SaveTokens(ctx context.Context, creds DiscordCredential) error {
	const query = `
		INSERT INTO auth_discord (
			steam_id, discord_id, access_token, refresh_token,
			expires_in, scope, token_type,  created_on, updated_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (steam_id)
		DO UPDATE SET discord_id = $2, access_token= $3, refresh_token = $4,
			expires_in = $5, scope = $6, token_type = $7, updated_on = $9`

	return database.DBErr(d.db.Exec(ctx, nil, query, creds.SteamID.Int64(), creds.DiscordID, creds.AccessToken, creds.RefreshToken,
		creds.ExpiresIn, creds.Scope, creds.TokenType, creds.CreatedOn, creds.UpdatedOn,
	))
}

func (d DiscordOAuthRepository) GetTokens(ctx context.Context, steamID steamid.SteamID) (DiscordCredential, error) {
	row, errRow := d.db.QueryRowBuilder(ctx, nil, d.db.Builder().
		Select("discord_id", "access_token", "refresh_token",
			"expires_in", "scope", "token_type", "created_on", "updated_on").
		From("auth_discord").
		Where(sq.Eq{"steam_id": steamID.Int64()}))
	if errRow != nil {
		return DiscordCredential{}, database.DBErr(errRow)
	}

	var creds DiscordCredential

	creds.SteamID = steamID

	if err := row.Scan(&creds.DiscordID, &creds.AccessToken, &creds.RefreshToken, &creds.ExpiresIn, &creds.Scope, &creds.TokenType,
		&creds.CreatedOn, &creds.UpdatedOn); err != nil {
		return DiscordCredential{}, database.DBErr(errRow)
	}

	return creds, nil
}

func (d DiscordOAuthRepository) DeleteTokens(ctx context.Context, steamID steamid.SteamID) error {
	query, vars, errQuery := d.db.Builder().
		Delete("auth_discord").
		Where(sq.Eq{"steam_id": steamID}).
		ToSql()
	if errQuery != nil {
		return database.DBErr(errQuery)
	}

	return d.db.Exec(ctx, nil, query, vars...)
}

func (d DiscordOAuthRepository) OldAuths(ctx context.Context) ([]DiscordCredential, error) {
	const query = `SELECT steam_id, discord_id, access_token,  refresh_token,
                          expires_in, scope, token_type, created_on, updated_on FROM auth_discord
					WHERE to_timestamp(extract(epoch from updated_on) + expires_in) < (now()) -- + interval '7 days');`

	rows, errRows := d.db.Query(ctx, nil, query)
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	var credentials []DiscordCredential

	for rows.Next() {
		var creds DiscordCredential
		if errScan := rows.Scan(&creds.SteamID, &creds.DiscordID, &creds.AccessToken, &creds.RefreshToken, &creds.ExpiresIn,
			&creds.Scope, &creds.TokenType, &creds.CreatedOn, &creds.UpdatedOn); errScan != nil {
			return credentials, database.DBErr(errScan)
		}

		credentials = append(credentials, creds)
	}

	return credentials, nil
}
