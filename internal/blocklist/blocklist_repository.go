package blocklist

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type blocklistRepository struct {
	db database.Database
}

func NewBlocklistRepository(database database.Database) domain.BlocklistRepository {
	return &blocklistRepository{db: database}
}

func (b *blocklistRepository) GetCIDRBlockSources(ctx context.Context) ([]domain.CIDRBlockSource, error) {
	blocks := make([]domain.CIDRBlockSource, 0)

	rows, errRows := b.db.QueryBuilder(ctx, b.db.
		Builder().
		Select("cidr_block_source_id", "name", "url", "enabled", "created_on", "updated_on").
		From("cidr_block_source"))
	if errRows != nil {
		if errors.Is(errRows, domain.ErrNoResult) {
			return blocks, nil
		}

		return nil, b.db.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var block domain.CIDRBlockSource
		if errScan := rows.Scan(&block.CIDRBlockSourceID, &block.Name, &block.URL, &block.Enabled, &block.CreatedOn, &block.UpdatedOn); errScan != nil {
			return nil, b.db.DBErr(errScan)
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}

func (b *blocklistRepository) GetCIDRBlockSource(ctx context.Context, sourceID int, block *domain.CIDRBlockSource) error {
	row, errRow := b.db.QueryRowBuilder(ctx, b.db.
		Builder().
		Select("cidr_block_source_id", "name", "url", "enabled", "created_on", "updated_on").
		From("cidr_block_source").
		Where(sq.Eq{"cidr_block_source_id": sourceID}))
	if errRow != nil {
		return b.db.DBErr(errRow)
	}

	if errScan := row.Scan(&block.CIDRBlockSourceID, &block.Name, &block.URL, &block.Enabled, &block.CreatedOn, &block.UpdatedOn); errScan != nil {
		return b.db.DBErr(errScan)
	}

	return nil
}

func (b *blocklistRepository) SaveCIDRBlockSources(ctx context.Context, block *domain.CIDRBlockSource) error {
	now := time.Now()

	block.UpdatedOn = now

	if block.CIDRBlockSourceID > 0 {
		return b.db.DBErr(b.db.ExecUpdateBuilder(ctx, b.db.
			Builder().
			Update("cidr_block_source").
			SetMap(map[string]interface{}{
				"name":       block.Name,
				"url":        block.URL,
				"enabled":    block.Enabled,
				"updated_on": block.UpdatedOn,
			}).
			Where(sq.Eq{"cidr_block_source_id": block.CIDRBlockSourceID})))
	}

	block.CreatedOn = now

	return b.db.DBErr(b.db.ExecInsertBuilderWithReturnValue(ctx, b.db.
		Builder().
		Insert("cidr_block_source").
		SetMap(map[string]interface{}{
			"name":       block.Name,
			"url":        block.URL,
			"enabled":    block.Enabled,
			"created_on": block.CreatedOn,
			"updated_on": block.UpdatedOn,
		}).
		Suffix("RETURNING cidr_block_source_id"), &block.CIDRBlockSourceID))
}

func (b *blocklistRepository) DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error {
	return b.db.DBErr(b.db.ExecDeleteBuilder(ctx, b.db.
		Builder().
		Delete("cidr_block_source").
		Where(sq.Eq{"cidr_block_source_id": blockSourceID})))
}

func (b *blocklistRepository) GetCIDRBlockWhitelists(ctx context.Context) ([]domain.CIDRBlockWhitelist, error) {
	whitelists := make([]domain.CIDRBlockWhitelist, 0)

	rows, errRows := b.db.QueryBuilder(ctx, b.db.
		Builder().
		Select("cidr_block_whitelist_id", "address", "created_on", "updated_on").
		From("cidr_block_whitelist"))
	if errRows != nil {
		if errors.Is(errRows, domain.ErrNoResult) {
			return whitelists, nil
		}

		return nil, b.db.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var whitelist domain.CIDRBlockWhitelist
		if errScan := rows.Scan(&whitelist.CIDRBlockWhitelistID, &whitelist.Address, &whitelist.CreatedOn, &whitelist.UpdatedOn); errScan != nil {
			return nil, b.db.DBErr(errScan)
		}

		whitelists = append(whitelists, whitelist)
	}

	return whitelists, nil
}

func (b *blocklistRepository) GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *domain.CIDRBlockWhitelist) error {
	rows, errRow := b.db.QueryRowBuilder(ctx, b.db.
		Builder().
		Select("cidr_block_whitelist_id", "address", "created_on", "updated_on").
		From("cidr_block_whitelist").
		Where(sq.Eq{"cidr_block_whitelist_id": whitelistID}))
	if errRow != nil {
		return b.db.DBErr(errRow)
	}

	if errScan := rows.Scan(&whitelist.CIDRBlockWhitelistID, &whitelist.Address, &whitelist.CreatedOn, &whitelist.UpdatedOn); errScan != nil {
		return b.db.DBErr(errScan)
	}

	return nil
}

func (b *blocklistRepository) SaveCIDRBlockWhitelist(ctx context.Context, whitelist *domain.CIDRBlockWhitelist) error {
	now := time.Now()

	whitelist.UpdatedOn = now

	if whitelist.CIDRBlockWhitelistID > 0 {
		return b.db.DBErr(b.db.ExecUpdateBuilder(ctx, b.db.
			Builder().
			Update("cidr_block_whitelist").
			SetMap(map[string]interface{}{
				"address":    whitelist.Address.String(),
				"updated_on": whitelist.UpdatedOn,
			})))
	}

	whitelist.CreatedOn = now

	return b.db.DBErr(b.db.ExecInsertBuilderWithReturnValue(ctx, b.db.
		Builder().
		Insert("cidr_block_whitelist").
		SetMap(map[string]interface{}{
			"address":    whitelist.Address.String(),
			"created_on": whitelist.CreatedOn,
			"updated_on": whitelist.UpdatedOn,
		}).
		Suffix("RETURNING cidr_block_whitelist_id"), &whitelist.CIDRBlockWhitelistID))
}

func (b *blocklistRepository) DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error {
	return b.db.DBErr(b.db.ExecDeleteBuilder(ctx, b.db.
		Builder().
		Delete("cidr_block_whitelist").
		Where(sq.Eq{"cidr_block_whitelist_id": whitelistID})))
}
