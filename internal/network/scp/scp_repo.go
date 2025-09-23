package scp

import (
	"context"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
)

// serverID is a unique game instance on a server.
type serverID struct {
	ServerID  int
	ShortName string
}

// ServerInfo represents a single *physical* machine. It can have several instances locally however as defined by multiple
// [erverID].
type ServerInfo struct {
	ServerIDs       []serverID
	Address         string
	AddressInternal string
}

func NewRepository(db database.Database) Repository {
	return Repository{db: db}
}

type Repository struct {
	db database.Database
}

func (r Repository) getKey(ctx context.Context, addr string) (string, error) {
	var key string

	if errRow := r.db.
		QueryRow(ctx, nil, `SELECT key FROM host_key WHERE address = $1`, addr).
		Scan(&key); errRow != nil {
		return "", database.DBErr(errRow)
	}

	return key, nil
}

func (r Repository) setKey(ctx context.Context, addr string, key string) error {
	const query = `INSERT INTO host_key (address, key, created_on) VALUES ($1, $2, $3)`
	if err := r.db.Exec(ctx, nil, query, addr, key, time.Now()); err != nil {
		return database.DBErr(err)
	}

	return nil
}

func (r Repository) Servers(ctx context.Context) ([]ServerInfo, error) {
	const query = `
		SELECT server_id, short_name, address, address_internal
		WHERE is_enabled = true AND deleted = false
		ORDER BY short_name`
	rows, errRows := r.db.Query(ctx, nil, query)
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}
	defer rows.Close()

	var (
		results []ServerInfo
		result  ServerInfo
		dirty   bool // Is there a result to still append on the last record
	)
	for rows.Next() {
		var (
			sid             serverID
			address         string
			addressInternal string
		)
		if err := rows.Scan(&sid.ServerID, &sid.ShortName, &address, &addressInternal); err != nil {
			return nil, database.DBErr(err)
		}

		if result.Address == "" {
			result = ServerInfo{ServerIDs: []serverID{sid}, Address: address, AddressInternal: addressInternal}
			dirty = true
		} else if result.Address != address {
			results = append(results, result)
			result = ServerInfo{ServerIDs: []serverID{sid}, Address: address, AddressInternal: addressInternal}
			dirty = false
		} else {
			result.ServerIDs = append(result.ServerIDs, sid)
			dirty = true
		}
	}

	if dirty {
		results = append(results, result)
	}

	return results, nil
}
