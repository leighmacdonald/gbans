package asn

import (
	"context"
	"net/netip"

	"github.com/leighmacdonald/gbans/internal/database"
)

func NewRepository(db database.Database) Repository {
	return Repository{db: db}
}

type Repository struct {
	db database.Database
}

func (r Repository) All(ctx context.Context) ([]Block, error) {
	rows, errRows := r.db.Query(ctx, `SELECT as_num, reason, notes, created_on, updated_on FROM as_num`)
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	defer rows.Close()

	var blocks []Block
	for rows.Next() {
		var block Block
		if err := rows.Scan(&block.ASNum, &block.Reason, &block.Notes, &block.CreatedOn, &block.UpdatedOn); err != nil {
			return nil, database.DBErr(err)
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}

func (r Repository) IsBlocked(_ context.Context, _ netip.Addr) bool {
	// const query = `SELECT as_num, ip_range FRON net_asn WHERE as_num = $1`

	return false
}

func (r Repository) Save(ctx context.Context, ban Block) error {
	const query = `
		INSERT INTO asn_ban (as_num, reason, notes, created_on, updated_on)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (as_num) DO UPDATE
		SET reason = $2, updated_on = $5`

	if err := r.db.Exec(ctx, query, ban.ASNum, ban.Reason, ban.Notes, ban.CreatedOn, ban.UpdatedOn); err != nil {
		return database.DBErr(err)
	}

	return nil
}

func (r Repository) Delete(ctx context.Context, asNum int) error {
	return database.DBErr(r.db.Exec(ctx, `DELETE FROM ban_asn WHERE as_num = $1`, asNum))
}
