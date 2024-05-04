package ban

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgtype"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type banNetRepository struct {
	db database.Database
}

func NewBanNetRepository(database database.Database) domain.BanNetRepository {
	return &banNetRepository{db: database}
}

// GetByAddress returns the BanCIDR matching intersecting the supplied ip.
//
// Note that this function does not currently limit results returned. This may change in the future, do not
// rely on this functionality.
func (r banNetRepository) GetByAddress(ctx context.Context, ipAddr netip.Addr) ([]domain.BanCIDR, error) {
	const query = `
		SELECT net_id, cidr, origin, created_on, updated_on, reason, reason_text, valid_until, deleted, 
		       note, unban_reason_text, is_enabled, target_id, source_id, appeal_state
		FROM ban_net
		WHERE $1 <<= cidr AND deleted = false AND is_enabled = true`

	var nets []domain.BanCIDR

	rows, errQuery := r.db.Query(ctx, query, ipAddr.String())
	if errQuery != nil {
		return nil, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			banNet   domain.BanCIDR
			sourceID int64
			targetID int64
			cidr     *net.IPNet
		)

		if errScan := rows.
			Scan(&banNet.NetID, &cidr, &banNet.Origin,
				&banNet.CreatedOn, &banNet.UpdatedOn, &banNet.Reason, &banNet.ReasonText,
				&banNet.ValidUntil, &banNet.Deleted, &banNet.Note, &banNet.UnbanReasonText,
				&banNet.IsEnabled, &targetID, &sourceID, &banNet.AppealState); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		banNet.CIDR = cidr.String()
		banNet.SourceID = steamid.New(sourceID)
		banNet.TargetID = steamid.New(targetID)

		nets = append(nets, banNet)
	}

	if nets == nil {
		return []domain.BanCIDR{}, nil
	}

	return nets, nil
}

// Get returns the BanCIDR matching intersecting the supplied ip.
func (r banNetRepository) Get(ctx context.Context, filter domain.CIDRBansQueryFilter) ([]domain.BannedCIDRPerson, int64, error) {
	validColumns := map[string][]string{
		"b.": {
			"net_id", "cidr", "origin", "created_on", "updated_on",
			"reason", "valid_until", "deleted", "is_enabled", "target_id", "source_id", "appeal_state",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}

	builder := r.db.
		Builder().
		Select("b.net_id", "b.cidr", "b.origin", "b.created_on", "b.updated_on",
			"b.reason", "b.reason_text", "b.valid_until", "b.deleted", "b.note", "b.unban_reason_text",
			"b.is_enabled", "b.target_id", "b.source_id", "b.appeal_state",
			"s.personaname as source_personaname", "s.avatarhash",
			"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans",
		).
		From("ban_net b").
		LeftJoin("person s ON s.steam_id = b.source_id").
		LeftJoin("person t ON t.steam_id = b.target_id")

	var constraints sq.And

	if !filter.Deleted {
		constraints = append(constraints, sq.Eq{"b.deleted": false})
	}

	if filter.Reason > 0 {
		constraints = append(constraints, sq.Eq{"b.reason": filter.Reason})
	}

	if filter.PermanentOnly {
		constraints = append(constraints, sq.Gt{"b.valid_until": time.Now()})
	}

	if sid, ok := filter.TargetSteamID(ctx); ok {
		constraints = append(constraints, sq.Eq{"b.target_id": sid})
	}

	if sid, ok := filter.SourceSteamID(ctx); ok {
		constraints = append(constraints, sq.Eq{"b.source_id": sid})
	}

	if filter.IP != "" {
		var addr string

		_, cidr, errCidr := net.ParseCIDR(filter.IP)

		if errCidr != nil {
			ip := net.ParseIP(filter.IP)
			if ip == nil {
				return nil, 0, errors.Join(errCidr, domain.ErrNetworkInvalidIP)
			}

			addr = ip.String()
		} else {
			addr = cidr.String()
		}

		constraints = append(constraints, sq.Expr("? <<= cidr", addr))
	}

	builder = filter.QueryFilter.ApplySafeOrder(builder, validColumns, "net_id")
	builder = filter.QueryFilter.ApplyLimitOffsetDefault(builder)

	var nets []domain.BannedCIDRPerson

	rows, errRows := r.db.QueryBuilder(ctx, builder.Where(constraints))
	if errRows != nil {
		return nil, 0, r.db.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			banNet   domain.BannedCIDRPerson
			sourceID int64
			targetID int64
			cidr     *net.IPNet
		)

		if errScan := rows.
			Scan(&banNet.NetID, &cidr, &banNet.Origin,
				&banNet.CreatedOn, &banNet.UpdatedOn, &banNet.Reason, &banNet.ReasonText,
				&banNet.ValidUntil, &banNet.Deleted, &banNet.Note, &banNet.UnbanReasonText,
				&banNet.IsEnabled, &targetID, &sourceID, &banNet.AppealState,
				&banNet.SourceTarget.SourcePersonaname, &banNet.SourceTarget.SourceAvatarhash,
				&banNet.SourceTarget.TargetPersonaname, &banNet.SourceTarget.TargetAvatarhash,
				&banNet.CommunityBanned, &banNet.VacBans, &banNet.GameBans); errScan != nil {
			return nil, 0, r.db.DBErr(errScan)
		}

		banNet.CIDR = cidr.String()
		banNet.SourceID = steamid.New(sourceID)
		banNet.TargetID = steamid.New(targetID)

		nets = append(nets, banNet)
	}

	count, errCount := r.db.GetCount(ctx, r.db.
		Builder().
		Select("COUNT(b.net_id)").
		From("ban_net b").
		Where(constraints))

	if errCount != nil {
		if errors.Is(errCount, domain.ErrNoResult) {
			return []domain.BannedCIDRPerson{}, 0, nil
		}

		return nil, count, r.db.DBErr(errCount)
	}

	if nets == nil {
		return []domain.BannedCIDRPerson{}, 0, nil
	}

	return nets, count, nil
}

func (r banNetRepository) updateBanNet(ctx context.Context, banNet *domain.BanCIDR) error {
	banNet.UpdatedOn = time.Now()

	query := r.db.
		Builder().
		Update("ban_net").
		Set("cidr", banNet.CIDR).
		Set("origin", banNet.Origin).
		Set("updated_on", banNet.UpdatedOn).
		Set("reason", banNet.Reason).
		Set("reason_text", banNet.ReasonText).
		Set("valid_until", banNet.ValidUntil).
		Set("deleted", banNet.Deleted).
		Set("note", banNet.Note).
		Set("unban_reason_text", banNet.UnbanReasonText).
		Set("is_enabled", banNet.IsEnabled).
		Set("target_id", banNet.TargetID.Int64()).
		Set("source_id", banNet.SourceID.Int64()).
		Set("appeal_state", banNet.AppealState).
		Where(sq.Eq{"net_id": banNet.NetID})

	return r.db.DBErr(r.db.ExecUpdateBuilder(ctx, query))
}

func (r banNetRepository) insertBanNet(ctx context.Context, banNet *domain.BanCIDR) error {
	query, args, errQueryArgs := r.db.
		Builder().
		Insert("ban_net").
		Columns("cidr", "origin", "created_on", "updated_on", "reason", "reason_text", "valid_until",
			"deleted", "note", "unban_reason_text", "is_enabled", "target_id", "source_id", "appeal_state").
		Values(banNet.CIDR, banNet.Origin, banNet.CreatedOn, banNet.UpdatedOn, banNet.Reason, banNet.ReasonText,
			banNet.ValidUntil, banNet.Deleted, banNet.Note, banNet.UnbanReasonText, banNet.IsEnabled,
			banNet.TargetID.Int64(), banNet.SourceID.Int64(), banNet.AppealState).
		Suffix("RETURNING net_id").
		ToSql()
	if errQueryArgs != nil {
		return r.db.DBErr(errQueryArgs)
	}

	return r.db.DBErr(r.db.QueryRow(ctx, query, args...).Scan(&banNet.NetID))
}

func (r banNetRepository) Save(ctx context.Context, banNet *domain.BanCIDR) error {
	if banNet.NetID > 0 {
		return r.updateBanNet(ctx, banNet)
	}

	return r.insertBanNet(ctx, banNet)
}

func (r banNetRepository) Delete(ctx context.Context, banNet *domain.BanCIDR) error {
	query := r.db.
		Builder().
		Delete("ban_net").
		Where(sq.Eq{"net_id": banNet.NetID})

	if errExec := r.db.ExecDeleteBuilder(ctx, query); errExec != nil {
		return r.db.DBErr(errExec)
	}

	banNet.NetID = 0

	return nil
}

func (r banNetRepository) Expired(ctx context.Context) ([]domain.BanCIDR, error) {
	query := r.db.
		Builder().
		Select("net_id", "cidr", "origin", "created_on", "updated_on", "reason_text", "valid_until",
			"deleted", "note", "unban_reason_text", "is_enabled", "target_id", "source_id", "reason", "appeal_state").
		From("ban_net").
		Where(sq.Lt{"valid_until": time.Now()})

	var bans []domain.BanCIDR

	rows, errQuery := r.db.QueryBuilder(ctx, query)
	if errQuery != nil {
		return nil, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			banNet   domain.BanCIDR
			targetID int64
			sourceID int64
			cidr     pgtype.CIDR
		)

		if errScan := rows.
			Scan(&banNet.NetID, &cidr, &banNet.Origin, &banNet.CreatedOn,
				&banNet.UpdatedOn, &banNet.ReasonText, &banNet.ValidUntil, &banNet.Deleted, &banNet.Note,
				&banNet.UnbanReasonText, &banNet.IsEnabled, &targetID, &sourceID,
				&banNet.Reason, &banNet.AppealState); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		banNet.CIDR = cidr.IPNet.String()
		banNet.TargetID = steamid.New(targetID)
		banNet.SourceID = steamid.New(sourceID)

		bans = append(bans, banNet)
	}

	if bans == nil {
		return []domain.BanCIDR{}, nil
	}

	return bans, nil
}

func (r banNetRepository) GetByID(ctx context.Context, netID int64, banNet *domain.BanCIDR) error {
	const query = `
		SELECT net_id, cidr, origin, created_on, updated_on, reason, reason_text, valid_until, deleted, 
		       note, unban_reason_text, is_enabled, target_id, source_id, appeal_state
		FROM ban_net
		WHERE deleted = false AND net_id = $1`

	var (
		sourceID int64
		targetID int64
		cidr     *net.IPNet
	)

	errQuery := r.db.
		QueryRow(ctx, query, netID).
		Scan(&banNet.NetID, &cidr, &banNet.Origin,
			&banNet.CreatedOn, &banNet.UpdatedOn, &banNet.Reason, &banNet.ReasonText,
			&banNet.ValidUntil, &banNet.Deleted, &banNet.Note, &banNet.UnbanReasonText,
			&banNet.IsEnabled, &targetID, &sourceID, &banNet.AppealState)
	if errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	banNet.CIDR = cidr.String()
	banNet.SourceID = steamid.New(sourceID)
	banNet.TargetID = steamid.New(targetID)

	return nil
}
