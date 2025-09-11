package auth

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type AuthRepository struct {
	db database.Database
}

func NewAuthRepository(database database.Database) *AuthRepository {
	return &AuthRepository{db: database}
}

func (r AuthRepository) SavePersonAuth(ctx context.Context, auth *PersonAuth) error {
	query, args, errQuery := r.db.
		Builder().
		Insert("person_auth").
		Columns("steam_id", "ip_addr", "refresh_token", "created_on").
		Values(auth.SteamID.Int64(), auth.IPAddr.String(), auth.AccessToken, auth.CreatedOn).
		Suffix("RETURNING \"person_auth_id\"").
		ToSql()

	if errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	return r.db.DBErr(r.db.QueryRow(ctx, nil, query, args...).Scan(&auth.PersonAuthID))
}

func (r AuthRepository) DeletePersonAuth(ctx context.Context, authID int64) error {
	return r.db.DBErr(r.db.ExecDeleteBuilder(ctx, nil, r.db.
		Builder().
		Delete("person_auth").
		Where(sq.Eq{"person_auth_id": authID})))
}

func (r AuthRepository) PrunePersonAuth(ctx context.Context) error {
	return r.db.DBErr(r.db.ExecDeleteBuilder(ctx, nil, r.db.
		Builder().
		Delete("person_auth").
		Where(sq.Gt{"created_on + interval '1 month'": time.Now()})))
}

func (r AuthRepository) GetPersonAuthByFingerprint(ctx context.Context, fingerprint string, auth *PersonAuth) error {
	row, errRow := r.db.QueryRowBuilder(ctx, nil, r.db.
		Builder().
		Select("person_auth_id", "steam_id", "ip_addr", "refresh_token", "created_on").
		From("person_auth").
		Where(sq.And{sq.Eq{"fingerprint": fingerprint}}))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	var steamID int64

	if errScan := row.Scan(&auth.PersonAuthID, &steamID, &auth.IPAddr, &auth.AccessToken, &auth.CreatedOn); errScan != nil {
		return r.db.DBErr(errScan)
	}

	auth.SteamID = steamid.New(steamID)

	return nil
}
