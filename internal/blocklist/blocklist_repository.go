package blocklist

import (
	"context"
	"errors"
	"net/netip"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type blocklistRepository struct {
	db database.Database
}

func NewBlocklistRepository(database database.Database) domain.BlocklistRepository {
	return &blocklistRepository{db: database}
}

func (b *blocklistRepository) InsertCache(ctx context.Context, list domain.CIDRBlockSource, entries []netip.Prefix) error {
	const query = "INSERT INTO cidr_block_entries (cidr_block_source_id, net_block, created_on) VALUES ($1, $2, $3)"

	batch := pgx.Batch{}
	now := time.Now()

	for _, cidrRange := range entries {
		batch.Queue(query, list.CIDRBlockSourceID, cidrRange, now)
	}

	batchResults := b.db.SendBatch(ctx, &batch)
	if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
		return errors.Join(errCloseBatch, domain.ErrCloseBatch)
	}

	return nil
}

func (b *blocklistRepository) TruncateCachedEntries(ctx context.Context) error {
	return b.db.DBErr(b.db.ExecDeleteBuilder(ctx, b.db.Builder().Delete("cidr_block_entries")))
}

func (b *blocklistRepository) CreateSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) (domain.WhitelistSteam, error) {
	now := time.Now()

	if err := b.db.ExecInsertBuilder(ctx, b.db.Builder().Insert("person_whitelist").SetMap(map[string]interface{}{
		"steam_id":   steamID.Int64(),
		"created_on": now,
		"updated_on": now,
	})); err != nil {
		return domain.WhitelistSteam{}, b.db.DBErr(err)
	}

	entry, errEntry := b.GetSteamBlockWhitelists(ctx)
	if errEntry != nil {
		return domain.WhitelistSteam{}, b.db.DBErr(errEntry)
	}

	for _, wl := range entry {
		if wl.SteamIDValue == steamID.String() {
			return wl, nil
		}
	}

	return domain.WhitelistSteam{}, domain.ErrInternal
}

func (b *blocklistRepository) GetSteamBlockWhitelists(ctx context.Context) ([]domain.WhitelistSteam, error) {
	blocks := make([]domain.WhitelistSteam, 0)

	rows, errRows := b.db.QueryBuilder(ctx, b.db.
		Builder().
		Select("w.steam_id", "p.personaname", "p.avatarhash", "w.created_on", "w.updated_on").
		From("person_whitelist w").
		LeftJoin("person p USING(steam_id)"))

	if errRows != nil {
		if errors.Is(errRows, domain.ErrNoResult) {
			return blocks, nil
		}

		return nil, b.db.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			block   domain.WhitelistSteam
			steamID int64
		)

		if errScan := rows.Scan(&steamID, &block.Personaname, &block.AvatarHash, &block.CreatedOn, &block.UpdatedOn); errScan != nil {
			return nil, b.db.DBErr(errScan)
		}

		sid := steamid.New(steamID)

		block.SteamIDValue = sid.String()

		blocks = append(blocks, block)
	}

	return blocks, nil
}

func (b *blocklistRepository) DeleteSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) error {
	return b.db.DBErr(b.db.ExecDeleteBuilder(ctx, b.db.
		Builder().
		Delete("person_whitelist").
		Where(sq.Eq{"steam_id": steamID.Int64()})))
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

func (b *blocklistRepository) GetCIDRBlockWhitelists(ctx context.Context) ([]domain.WhitelistIP, error) {
	whitelists := make([]domain.WhitelistIP, 0)

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
		var whitelist domain.WhitelistIP
		if errScan := rows.Scan(&whitelist.CIDRBlockWhitelistID, &whitelist.Address, &whitelist.CreatedOn, &whitelist.UpdatedOn); errScan != nil {
			return nil, b.db.DBErr(errScan)
		}

		whitelists = append(whitelists, whitelist)
	}

	return whitelists, nil
}

func (b *blocklistRepository) GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *domain.WhitelistIP) error {
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

func (b *blocklistRepository) SaveCIDRBlockWhitelist(ctx context.Context, whitelist *domain.WhitelistIP) error {
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
