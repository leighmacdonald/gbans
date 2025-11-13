package auth

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct {
	database.Database
}

func NewRepository(database database.Database) Repository {
	return Repository{Database: database}
}

func (r Repository) SavePersonAuth(ctx context.Context, auth *PersonAuth) error {
	query, args, errQuery := r.Builder().
		Insert("person_auth").
		Columns("steam_id", "ip_addr", "refresh_token", "created_on").
		Values(auth.SteamID.Int64(), auth.IPAddr.String(), auth.AccessToken, auth.CreatedOn).
		Suffix("RETURNING \"person_auth_id\"").
		ToSql()

	if errQuery != nil {
		return database.DBErr(errQuery)
	}

	return database.DBErr(r.QueryRow(ctx, query, args...).Scan(&auth.PersonAuthID))
}

func (r Repository) DeletePersonAuth(ctx context.Context, authID int64) error {
	return database.DBErr(r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("person_auth").
		Where(sq.Eq{"person_auth_id": authID})))
}

func (r Repository) PrunePersonAuth(ctx context.Context) error {
	return database.DBErr(r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("person_auth").
		Where(sq.Gt{"created_on + interval '1 month'": time.Now()})))
}

func (r Repository) GetPersonAuthByFingerprint(ctx context.Context, fingerprint string, auth *PersonAuth) error {
	row, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select("person_auth_id", "steam_id", "ip_addr", "refresh_token", "created_on").
		From("person_auth").
		Where(sq.And{sq.Eq{"fingerprint": fingerprint}}))
	if errRow != nil {
		return database.DBErr(errRow)
	}

	var steamID int64

	if errScan := row.Scan(&auth.PersonAuthID, &steamID, &auth.IPAddr, &auth.AccessToken, &auth.CreatedOn); errScan != nil {
		return database.DBErr(errScan)
	}

	auth.SteamID = steamid.New(steamID)

	return nil
}
