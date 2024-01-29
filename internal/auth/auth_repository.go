package auth

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type authRepository struct {
	db database.Database
}

func NewAuthRepository(database database.Database) domain.AuthRepository {
	return &authRepository{db: database}
}

func (r authRepository) SavePersonAuth(ctx context.Context, auth *domain.PersonAuth) error {
	query, args, errQuery := r.db.
		Builder().
		Insert("person_auth").
		Columns("steam_id", "ip_addr", "refresh_token", "created_on").
		Values(auth.SteamID.Int64(), auth.IPAddr.String(), auth.RefreshToken, auth.CreatedOn).
		Suffix("RETURNING \"person_auth_id\"").
		ToSql()

	if errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	return r.db.DBErr(r.db.QueryRow(ctx, query, args...).Scan(&auth.PersonAuthID))
}

func (r authRepository) DeletePersonAuth(ctx context.Context, authID int64) error {
	return r.db.DBErr(r.db.ExecDeleteBuilder(ctx, r.db.
		Builder().
		Delete("person_auth").
		Where(sq.Eq{"person_auth_id": authID})))
}

func (r authRepository) PrunePersonAuth(ctx context.Context) error {
	return r.db.DBErr(r.db.ExecDeleteBuilder(ctx, r.db.
		Builder().
		Delete("person_auth").
		Where(sq.Gt{"created_on + interval '1 month'": time.Now()})))
}

func (r authRepository) GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *domain.PersonAuth) error {
	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("person_auth_id", "steam_id", "ip_addr", "refresh_token", "created_on").
		From("person_auth").
		Where(sq.And{sq.Eq{"refresh_token": token}}))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	var steamID int64

	if errScan := row.Scan(&auth.PersonAuthID, &steamID, &auth.IPAddr, &auth.RefreshToken, &auth.CreatedOn); errScan != nil {
		return r.db.DBErr(errScan)
	}

	auth.SteamID = steamid.New(steamID)

	return nil
}
