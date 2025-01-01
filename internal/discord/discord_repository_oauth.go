package discord

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type discordOAuthRepository struct {
	db database.Database
}

func NewDiscordOAuthRepository(database database.Database) domain.DiscordOAuthRepository {
	return &discordOAuthRepository{db: database}
}

func (d discordOAuthRepository) SaveUserDetail(ctx context.Context, detail domain.DiscordUserDetail) error {
	const query = `INSERT INTO discord_user (
                          steam_id, discord_id, username, avatar,
                          publicflags, mfa_enabled, premium_type, created_on, updated_on)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				ON CONFLICT (discord_id)
				DO UPDATE SET discord_id = $2, username= $3, avatar = $4,
				              publicflags = $5, mfa_enabled = $6, premium_type = $7, updated_on = $9
				`

	return d.db.DBErr(d.db.Exec(ctx, nil, query, detail.SteamID.Int64(), detail.ID, detail.Username, detail.Avatar,
		detail.PublicFlags, detail.MfaEnabled, detail.PremiumType, detail.CreatedOn, detail.UpdatedOn,
	))
}

func (d discordOAuthRepository) GetUserDetail(ctx context.Context, steamID steamid.SteamID) (domain.DiscordUserDetail, error) {
	row, errRow := d.db.QueryRowBuilder(ctx, nil, d.db.Builder().
		Select("discord_id", "username", "avatar",
			"publicflags", "mfa_enabled", "premium_type", "created_on", "updated_on").
		From("discord_user").
		Where(sq.Eq{"steam_id": steamID.Int64()}))
	if errRow != nil {
		return domain.DiscordUserDetail{}, d.db.DBErr(errRow)
	}

	var detail domain.DiscordUserDetail

	detail.SteamID = steamID

	if err := row.Scan(&detail.ID, &detail.Username, &detail.Avatar, &detail.PublicFlags, &detail.MfaEnabled, &detail.PremiumType,
		&detail.CreatedOn, &detail.UpdatedOn); err != nil {
		return domain.DiscordUserDetail{}, d.db.DBErr(errRow)
	}

	return detail, nil
}

func (d discordOAuthRepository) DeleteUserDetail(ctx context.Context, steamID steamid.SteamID) error {
	return d.db.DBErr(d.db.ExecDeleteBuilder(ctx, nil, d.db.Builder().
		Delete("discord_user").
		Where(sq.Eq{"steam_id": steamID})))
}

func (d discordOAuthRepository) SaveTokens(ctx context.Context, creds domain.DiscordCredential) error {
	const query = `
		INSERT INTO auth_discord (
			steam_id, discord_id, access_token, refresh_token,
			expires_in, scope, token_type,  created_on, updated_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (steam_id)
		DO UPDATE SET discord_id = $2, access_token= $3, refresh_token = $4,
			expires_in = $5, scope = $6, token_type = $7, updated_on = $9`

	return d.db.DBErr(d.db.Exec(ctx, nil, query, creds.SteamID.Int64(), creds.DiscordID, creds.AccessToken, creds.RefreshToken,
		creds.ExpiresIn, creds.Scope, creds.TokenType, creds.CreatedOn, creds.UpdatedOn,
	))
}

func (d discordOAuthRepository) GetTokens(ctx context.Context, steamID steamid.SteamID) (domain.DiscordCredential, error) {
	row, errRow := d.db.QueryRowBuilder(ctx, nil, d.db.Builder().
		Select("discord_id", "access_token", "refresh_token",
			"expires_in", "scope", "token_type", "created_on", "updated_on").
		From("auth_discord").
		Where(sq.Eq{"steam_id": steamID.Int64()}))
	if errRow != nil {
		return domain.DiscordCredential{}, d.db.DBErr(errRow)
	}

	var creds domain.DiscordCredential

	creds.SteamID = steamID

	if err := row.Scan(&creds.DiscordID, &creds.AccessToken, &creds.RefreshToken, &creds.ExpiresIn, &creds.Scope, &creds.TokenType,
		&creds.CreatedOn, &creds.UpdatedOn); err != nil {
		return domain.DiscordCredential{}, d.db.DBErr(errRow)
	}

	return creds, nil
}

func (d discordOAuthRepository) DeleteTokens(ctx context.Context, steamID steamid.SteamID) error {
	query, vars, errQuery := d.db.Builder().
		Delete("auth_discord").
		Where(sq.Eq{"steam_id": steamID}).
		ToSql()
	if errQuery != nil {
		return d.db.DBErr(errQuery)
	}

	return d.db.Exec(ctx, nil, query, vars...)
}

func (d discordOAuthRepository) OldAuths(ctx context.Context) ([]domain.DiscordCredential, error) {
	const query = `SELECT steam_id, discord_id, access_token,  refresh_token,
                          expires_in, scope, token_type, created_on, updated_on FROM auth_discord
					WHERE to_timestamp(extract(epoch from updated_on) + expires_in) < (now()) -- + interval '7 days');`

	rows, errRows := d.db.Query(ctx, nil, query)
	if errRows != nil {
		return nil, d.db.DBErr(errRows)
	}

	var credentials []domain.DiscordCredential

	for rows.Next() {
		var creds domain.DiscordCredential
		if errScan := rows.Scan(&creds.SteamID, &creds.DiscordID, &creds.AccessToken, &creds.RefreshToken, &creds.ExpiresIn,
			&creds.Scope, &creds.TokenType, &creds.CreatedOn, &creds.UpdatedOn); errScan != nil {
			return credentials, d.db.DBErr(errScan)
		}

		credentials = append(credentials, creds)
	}

	return credentials, nil
}
