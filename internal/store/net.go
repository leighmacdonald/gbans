package store

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

var (
	ErrScanASN    = errors.New("failed to scan asn result")
	ErrCloseBatch = errors.New("failed to close batch request")
)

// GetBanNetByAddress returns the BanCIDR matching intersecting the supplied ip.
//
// Note that this function does not currently limit results returned. This may change in the future, do not
// rely on this functionality.
func (s Stores) GetBanNetByAddress(ctx context.Context, ipAddr net.IP) ([]model.BanCIDR, error) {
	const query = `
		SELECT net_id, cidr, origin, created_on, updated_on, reason, reason_text, valid_until, deleted, 
		       note, unban_reason_text, is_enabled, target_id, source_id, appeal_state
		FROM ban_net
		WHERE $1 <<= cidr AND deleted = false AND is_enabled = true`

	var nets []model.BanCIDR

	rows, errQuery := s.Query(ctx, query, ipAddr.String())
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			banNet   model.BanCIDR
			sourceID int64
			targetID int64
			cidr     *net.IPNet
		)

		if errScan := rows.
			Scan(&banNet.NetID, &cidr, &banNet.Origin,
				&banNet.CreatedOn, &banNet.UpdatedOn, &banNet.Reason, &banNet.ReasonText,
				&banNet.ValidUntil, &banNet.Deleted, &banNet.Note, &banNet.UnbanReasonText,
				&banNet.IsEnabled, &targetID, &sourceID, &banNet.AppealState); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		banNet.CIDR = cidr.String()
		banNet.SourceID = steamid.New(sourceID)
		banNet.TargetID = steamid.New(targetID)

		nets = append(nets, banNet)
	}

	if nets == nil {
		return []model.BanCIDR{}, nil
	}

	return nets, nil
}

func (s Stores) GetBanNetByID(ctx context.Context, netID int64, banNet *model.BanCIDR) error {
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

	errQuery := s.
		QueryRow(ctx, query, netID).
		Scan(&banNet.NetID, &cidr, &banNet.Origin,
			&banNet.CreatedOn, &banNet.UpdatedOn, &banNet.Reason, &banNet.ReasonText,
			&banNet.ValidUntil, &banNet.Deleted, &banNet.Note, &banNet.UnbanReasonText,
			&banNet.IsEnabled, &targetID, &sourceID, &banNet.AppealState)
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	banNet.CIDR = cidr.String()
	banNet.SourceID = steamid.New(sourceID)
	banNet.TargetID = steamid.New(targetID)

	return nil
}

func (s Stores) GetPlayerMostRecentIP(ctx context.Context, steamID steamid.SID64) net.IP {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("c.ip_addr").
		From("person_connections c").
		Where(sq.Eq{"c.steam_id": steamID.Int64()}).
		OrderBy("c.created_on desc").
		Limit(1))
	if errRow != nil {
		if errors.Is(errRow, errs.ErrNoResult) {
			return nil
		}

		return nil
	}

	var addr net.IP
	if errScan := row.Scan(&addr); errScan != nil {
		return nil
	}

	return addr
}

// GetBansNet returns the BanCIDR matching intersecting the supplied ip.
func (s Stores) GetBansNet(ctx context.Context, filter model.CIDRBansQueryFilter) ([]model.BannedCIDRPerson, int64, error) {
	validColumns := map[string][]string{
		"b.": {
			"net_id", "cidr", "origin", "created_on", "updated_on",
			"reason", "valid_until", "deleted", "is_enabled", "target_id", "source_id", "appeal_state",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}

	builder := s.
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

	if filter.TargetID != "" {
		targetID, errTargetID := filter.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Join(errTargetID, errs.ErrTargetID)
		}

		constraints = append(constraints, sq.Eq{"b.target_id": targetID.Int64()})
	}

	if filter.SourceID != "" {
		sourceID, errSourceID := filter.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errors.Join(errSourceID, errs.ErrSourceID)
		}

		constraints = append(constraints, sq.Eq{"b.source_id": sourceID.Int64()})
	}

	if filter.IP != "" {
		var addr string

		_, cidr, errCidr := net.ParseCIDR(filter.IP)

		if errCidr != nil {
			ip := net.ParseIP(filter.IP)
			if ip == nil {
				return nil, 0, errors.Join(errCidr, errs.ErrInvalidIP)
			}

			addr = ip.String()
		} else {
			addr = cidr.String()
		}

		constraints = append(constraints, sq.Expr("? <<= cidr", addr))
	}

	builder = filter.QueryFilter.ApplySafeOrder(builder, validColumns, "net_id")
	builder = filter.QueryFilter.ApplyLimitOffsetDefault(builder)

	var nets []model.BannedCIDRPerson

	rows, errRows := s.QueryBuilder(ctx, builder.Where(constraints))
	if errRows != nil {
		return nil, 0, errs.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			banNet   model.BannedCIDRPerson
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
			return nil, 0, errs.DBErr(errScan)
		}

		banNet.CIDR = cidr.String()
		banNet.SourceID = steamid.New(sourceID)
		banNet.TargetID = steamid.New(targetID)

		nets = append(nets, banNet)
	}

	count, errCount := getCount(ctx, s, s.
		Builder().
		Select("COUNT(b.net_id)").
		From("ban_net b").
		Where(constraints))

	if errCount != nil {
		if errors.Is(errCount, errs.ErrNoResult) {
			return []model.BannedCIDRPerson{}, 0, nil
		}

		return nil, count, errs.DBErr(errCount)
	}

	if nets == nil {
		return []model.BannedCIDRPerson{}, 0, nil
	}

	return nets, count, nil
}

func (s Stores) updateBanNet(ctx context.Context, banNet *model.BanCIDR) error {
	banNet.UpdatedOn = time.Now()

	query := s.
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

	return errs.DBErr(s.ExecUpdateBuilder(ctx, query))
}

func (s Stores) insertBanNet(ctx context.Context, banNet *model.BanCIDR) error {
	query, args, errQueryArgs := s.
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
		return errs.DBErr(errQueryArgs)
	}

	return errs.DBErr(s.QueryRow(ctx, query, args...).Scan(&banNet.NetID))
}

func (s Stores) SaveBanNet(ctx context.Context, banNet *model.BanCIDR) error {
	if banNet.NetID > 0 {
		return s.updateBanNet(ctx, banNet)
	}

	return s.insertBanNet(ctx, banNet)
}

func (s Stores) DropBanNet(ctx context.Context, banNet *model.BanCIDR) error {
	query := s.
		Builder().
		Delete("ban_net").
		Where(sq.Eq{"net_id": banNet.NetID})

	if errExec := s.ExecDeleteBuilder(ctx, query); errExec != nil {
		return errs.DBErr(errExec)
	}

	banNet.NetID = 0

	return nil
}

func (s Stores) GetExpiredNetBans(ctx context.Context) ([]model.BanCIDR, error) {
	query := s.
		Builder().
		Select("net_id", "cidr", "origin", "created_on", "updated_on", "reason_text", "valid_until",
			"deleted", "note", "unban_reason_text", "is_enabled", "target_id", "source_id", "reason", "appeal_state").
		From("ban_net").
		Where(sq.Lt{"valid_until": time.Now()})

	var bans []model.BanCIDR

	rows, errQuery := s.QueryBuilder(ctx, query)
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			banNet   model.BanCIDR
			targetID int64
			sourceID int64
			cidr     pgtype.CIDR
		)

		if errScan := rows.
			Scan(&banNet.NetID, &cidr, &banNet.Origin, &banNet.CreatedOn,
				&banNet.UpdatedOn, &banNet.ReasonText, &banNet.ValidUntil, &banNet.Deleted, &banNet.Note,
				&banNet.UnbanReasonText, &banNet.IsEnabled, &targetID, &sourceID,
				&banNet.Reason, &banNet.AppealState); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		banNet.CIDR = cidr.IPNet.String()
		banNet.TargetID = steamid.New(targetID)
		banNet.SourceID = steamid.New(sourceID)

		bans = append(bans, banNet)
	}

	if bans == nil {
		return []model.BanCIDR{}, nil
	}

	return bans, nil
}

func (s Stores) GetExpiredASNBans(ctx context.Context) ([]model.BanASN, error) {
	query := s.
		Builder().
		Select("ban_asn_id", "as_num", "origin", "source_id", "target_id", "reason_text", "valid_until",
			"created_on", "updated_on", "deleted", "reason", "is_enabled", "unban_reason_text", "appeal_state").
		From("ban_asn").
		Where(sq.And{sq.Lt{"valid_until": time.Now()}, sq.Eq{"deleted": false}})

	var bans []model.BanASN

	rows, errQuery := s.QueryBuilder(ctx, query)
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			banASN   model.BanASN
			targetID int64
			sourceID int64
		)

		if errScan := rows.
			Scan(&banASN.BanASNId, &banASN.ASNum, &banASN.Origin, &sourceID, &targetID,
				&banASN.ReasonText, &banASN.ValidUntil, &banASN.CreatedOn, &banASN.UpdatedOn, &banASN.Deleted,
				&banASN.Reason, &banASN.IsEnabled, &banASN.UnbanReasonText, &banASN.AppealState); errScan != nil {
			return nil, errors.Join(errScan, ErrScanASN)
		}

		banASN.TargetID = steamid.New(targetID)
		banASN.SourceID = steamid.New(sourceID)

		bans = append(bans, banASN)
	}

	if bans == nil {
		bans = []model.BanASN{}
	}

	return bans, nil
}

func (s Stores) GetASNRecordsByNum(ctx context.Context, asNum int64) (ip2location.ASNRecords, error) {
	query := s.
		Builder().
		Select("ip_from", "ip_to", "cidr", "as_num", "as_name").
		From("net_asn").
		Where(sq.Eq{"as_num": asNum})

	rows, errQuery := s.QueryBuilder(ctx, query)
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	records := ip2location.ASNRecords{}

	for rows.Next() {
		var asnRecord ip2location.ASNRecord
		if errScan := rows.
			Scan(&asnRecord.IPFrom, &asnRecord.IPTo, &asnRecord.CIDR, &asnRecord.ASNum, &asnRecord.ASName); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		records = append(records, asnRecord)
	}

	return records, nil
}

func (s Stores) GetASNRecordByIP(ctx context.Context, ipAddr net.IP, asnRecord *ip2location.ASNRecord) error {
	const query = `
		SELECT ip_from, ip_to, cidr, as_num, as_name 
		FROM net_asn
		WHERE $1::inet <@ ip_range
		LIMIT 1`

	if errQuery := s.
		QueryRow(ctx, query, ipAddr.String()).
		Scan(&asnRecord.IPFrom, &asnRecord.IPTo, &asnRecord.CIDR, &asnRecord.ASNum, &asnRecord.ASName); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) GetLocationRecord(ctx context.Context, ipAddr net.IP, record *ip2location.LocationRecord) error {
	const query = `
		SELECT ip_from, ip_to, country_code, country_name, region_name, city_name, ST_Y(location), ST_X(location) 
		FROM net_location 
		WHERE ip_range @> $1::inet`

	if errQuery := s.QueryRow(ctx, query, ipAddr.String()).
		Scan(&record.IPFrom, &record.IPTo, &record.CountryCode, &record.CountryName, &record.RegionName,
			&record.CityName, &record.LatLong.Latitude, &record.LatLong.Longitude); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) GetProxyRecord(ctx context.Context, ipAddr net.IP, proxyRecord *ip2location.ProxyRecord) error {
	const query = `
		SELECT ip_from, ip_to, proxy_type, country_code, country_name, region_name, 
       		city_name, isp, domain_used, usage_type, as_num, as_name, last_seen, threat 
		FROM net_proxy 
		WHERE $1::inet <@ ip_range`

	if errQuery := s.QueryRow(ctx, query, ipAddr.String()).
		Scan(&proxyRecord.IPFrom, &proxyRecord.IPTo, &proxyRecord.ProxyType, &proxyRecord.CountryCode, &proxyRecord.CountryName, &proxyRecord.RegionName, &proxyRecord.CityName, &proxyRecord.ISP,
			&proxyRecord.Domain, &proxyRecord.UsageType, &proxyRecord.ASN, &proxyRecord.AS, &proxyRecord.LastSeen, &proxyRecord.Threat); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) loadASN(ctx context.Context, log *zap.Logger, records []ip2location.ASNRecord) error {
	curTime := time.Now()

	if errTruncate := truncateTable(ctx, s, "net_asn"); errTruncate != nil {
		return errTruncate
	}

	const query = `
		INSERT INTO net_asn (ip_from, ip_to, cidr, as_num, as_name, ip_range) 
		VALUES($1, $2, $3, $4, $5, iprange($1, $2))`

	batch := pgx.Batch{}

	for recordIdx, asnRecord := range records {
		batch.Queue(query, asnRecord.IPFrom, asnRecord.IPTo, asnRecord.CIDR, asnRecord.ASNum, asnRecord.ASName)

		if recordIdx > 0 && recordIdx%100000 == 0 || len(records) == recordIdx+1 {
			if batch.Len() > 0 {
				c, cancel := context.WithTimeout(ctx, time.Second*10)

				batchResults := s.SendBatch(c, &batch)
				if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
					cancel()

					return errors.Join(errCloseBatch, ErrCloseBatch)
				}

				cancel()

				batch = pgx.Batch{}

				log.Info(fmt.Sprintf("ASN Progress: %d/%d (%.0f%%)",
					recordIdx, len(records)-1, float64(recordIdx)/float64(len(records)-1)*100))
			}
		}
	}

	log.Info("Loaded ASN4 records",
		zap.Int("count", len(records)), zap.Duration("duration", time.Since(curTime)))

	return nil
}

func (s Stores) loadLocation(ctx context.Context, log *zap.Logger, records []ip2location.LocationRecord, _ bool) error {
	curTime := time.Now()

	if errTruncate := truncateTable(ctx, s, "net_location"); errTruncate != nil {
		return errTruncate
	}

	const query = `
		INSERT INTO net_location (ip_from, ip_to, country_code, country_name, region_name, city_name, location, ip_range)
		VALUES($1, $2, $3, $4, $5, $6, ST_SetSRID(ST_MakePoint($8, $7), 4326), iprange($1, $2))`

	batch := pgx.Batch{}

	for recordIdx, locationRecord := range records {
		batch.Queue(query, locationRecord.IPFrom, locationRecord.IPTo, locationRecord.CountryCode, locationRecord.CountryName, locationRecord.RegionName, locationRecord.CityName, locationRecord.LatLong.Latitude, locationRecord.LatLong.Longitude)

		if recordIdx > 0 && recordIdx%100000 == 0 || len(records) == recordIdx+1 {
			if batch.Len() > 0 {
				c, cancel := context.WithTimeout(ctx, time.Second*10)

				batchResults := s.SendBatch(c, &batch)
				if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
					cancel()

					return errors.Join(errCloseBatch, ErrCloseBatch)
				}

				cancel()

				batch = pgx.Batch{}

				log.Info(fmt.Sprintf("Location4 Progress: %d/%d (%.0f%%)",
					recordIdx, len(records)-1, float64(recordIdx)/float64(len(records)-1)*100))
			}
		}
	}

	log.Info("Loaded Location4 records",
		zap.Int("count", len(records)), zap.Duration("duration", time.Since(curTime)))

	return nil
}

func (s Stores) loadProxies(ctx context.Context, log *zap.Logger, records []ip2location.ProxyRecord, _ bool) error {
	curTime := time.Now()

	if errTruncate := truncateTable(ctx, s, "net_proxy"); errTruncate != nil {
		return errTruncate
	}

	const query = `
		INSERT INTO net_proxy (ip_from, ip_to, proxy_type, country_code, country_name, region_name, city_name, isp,
		                       domain_used, usage_type, as_num, as_name, last_seen, threat, ip_range)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, iprange($1, $2))`

	batch := pgx.Batch{}

	for recordIdx, proxyRecord := range records {
		batch.Queue(query, proxyRecord.IPFrom, proxyRecord.IPTo, proxyRecord.ProxyType, proxyRecord.CountryCode, proxyRecord.CountryName, proxyRecord.RegionName, proxyRecord.CityName,
			proxyRecord.ISP, proxyRecord.Domain, proxyRecord.UsageType, proxyRecord.ASN, proxyRecord.AS, proxyRecord.LastSeen, proxyRecord.Threat)

		if recordIdx > 0 && recordIdx%100000 == 0 || len(records) == recordIdx+1 {
			if batch.Len() > 0 {
				c, cancel := context.WithTimeout(ctx, time.Second*120)

				batchResults := s.SendBatch(c, &batch)
				if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
					cancel()

					return errors.Join(errCloseBatch, ErrCloseBatch)
				}

				cancel()

				batch = pgx.Batch{}

				log.Info(fmt.Sprintf("Proxy Progress: %d/%d (%.0f%%)",
					recordIdx, len(records)-1, float64(recordIdx)/float64(len(records)-1)*100))
			}
		}
	}

	log.Info("Loaded Proxy records",
		zap.Int("count", len(records)), zap.Duration("duration", time.Since(curTime)))

	return nil
}

// InsertBlockListData will load the provided datasets into the database
//
// Note that this can take a while on slower machines. For reference, it takes
// about ~90s with a local database on a Ryzen 3900X/PCIe4 NVMe SSD.
func (s Stores) InsertBlockListData(ctx context.Context, log *zap.Logger, blockListData *ip2location.BlockListData) error {
	if len(blockListData.Proxies) > 0 {
		if errProxies := s.loadProxies(ctx, log, blockListData.Proxies, false); errProxies != nil {
			return errProxies
		}
	}

	if len(blockListData.Locations4) > 0 {
		if errLocation := s.loadLocation(ctx, log, blockListData.Locations4, false); errLocation != nil {
			return errLocation
		}
	}

	if len(blockListData.ASN4) > 0 {
		if errASN := s.loadASN(ctx, log, blockListData.ASN4); errASN != nil {
			return errASN
		}
	}

	return nil
}

func (s Stores) GetBanASN(ctx context.Context, asNum int64, banASN *model.BanASN) error {
	const query = `
		SELECT ban_asn_id, as_num, origin, source_id, target_id, reason_text, valid_until, created_on, updated_on, 
		       deleted, reason, is_enabled, unban_reason_text, appeal_state
		FROM ban_asn 
		WHERE deleted = false AND as_num = $1`

	var (
		targetID int64
		sourceID int64
	)

	if errQuery := s.
		QueryRow(ctx, query, asNum).
		Scan(&banASN.BanASNId, &banASN.ASNum, &banASN.Origin,
			&sourceID, &targetID, &banASN.ReasonText, &banASN.ValidUntil, &banASN.CreatedOn,
			&banASN.UpdatedOn, &banASN.Deleted, &banASN.Reason, &banASN.IsEnabled, &banASN.UnbanReasonText,
			&banASN.AppealState); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	banASN.TargetID = steamid.New(targetID)
	banASN.SourceID = steamid.New(sourceID)

	return nil
}

func (s Stores) GetBansASN(ctx context.Context, filter model.ASNBansQueryFilter) ([]model.BannedASNPerson, int64, error) {
	builder := s.
		Builder().
		Select("b.ban_asn_id", "b.as_num", "b.origin", "b.source_id",
			"b.target_id", "b.reason_text", "b.valid_until", "b.created_on", "b.updated_on",
			"b.deleted", "b.reason", "b.is_enabled", "b.unban_reason_text", "b.appeal_state",
			"coalesce(s.personaname, '') as source_personaname", "coalesce(s.avatarhash, '')",
			"coalesce(t.personaname, '') as target_personaname", "coalesce(t.avatarhash, '')",
			"coalesce(t.community_banned, false)", "coalesce(t.vac_bans, 0)", "coalesce(t.game_bans, 0)").
		From("ban_asn b").
		LeftJoin("person s on s.steam_id = b.source_id").
		LeftJoin("person t on t.steam_id = b.target_id")

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

	if filter.TargetID != "" {
		targetID, errTargetID := filter.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Join(errTargetID, errs.ErrTargetID)
		}

		constraints = append(constraints, sq.Eq{"b.target_id": targetID.Int64()})
	}

	if filter.SourceID != "" {
		sourceID, errSourceID := filter.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errors.Join(errSourceID, errs.ErrSourceID)
		}

		constraints = append(constraints, sq.Eq{"b.source_id": sourceID.Int64()})
	}

	if filter.ASNum > 0 {
		constraints = append(constraints, sq.Eq{"b.as_num": filter.ASNum})
	}

	builder = filter.QueryFilter.ApplySafeOrder(builder, map[string][]string{
		"b.": {
			"ban_asn_id", "as_num", "origin", "source_id", "target_id", "valid_until", "created_on", "updated_on",
			"deleted", "reason", "is_enabled", "appeal_state",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}, "ban_asn_id")

	builder = filter.QueryFilter.ApplyLimitOffsetDefault(builder)

	rows, errRows := s.QueryBuilder(ctx, builder.Where(constraints))
	if errRows != nil {
		if errors.Is(errRows, errs.ErrNoResult) {
			return []model.BannedASNPerson{}, 0, nil
		}

		return nil, 0, errs.DBErr(errRows)
	}

	defer rows.Close()

	var records []model.BannedASNPerson

	for rows.Next() {
		var (
			ban      model.BannedASNPerson
			targetID int64
			sourceID int64
		)

		if errScan := rows.
			Scan(&ban.BanASNId, &ban.ASNum, &ban.Origin, &sourceID, &targetID, &ban.ReasonText, &ban.ValidUntil,
				&ban.CreatedOn, &ban.UpdatedOn, &ban.Deleted, &ban.Reason, &ban.IsEnabled,
				&ban.UnbanReasonText, &ban.AppealState,
				&ban.SourceTarget.SourcePersonaname, &ban.SourceTarget.SourceAvatarhash,
				&ban.SourceTarget.TargetPersonaname, &ban.SourceTarget.TargetAvatarhash,
				&ban.CommunityBanned, &ban.VacBans, &ban.GameBans); errScan != nil {
			return nil, 0, errs.DBErr(errScan)
		}

		ban.SourceID = steamid.New(sourceID)
		ban.TargetID = steamid.New(targetID)

		records = append(records, ban)
	}

	count, errCount := getCount(ctx, s, s.
		Builder().
		Select("COUNT(b.ban_asn_id)").
		From("ban_asn b").
		Where(constraints))

	if errCount != nil {
		if errors.Is(errCount, errs.ErrNoResult) {
			return []model.BannedASNPerson{}, 0, nil
		}

		return nil, 0, errs.DBErr(errCount)
	}

	if records == nil {
		records = []model.BannedASNPerson{}
	}

	return records, count, nil
}

func (s Stores) SaveBanASN(ctx context.Context, banASN *model.BanASN) error {
	banASN.UpdatedOn = time.Now()

	if banASN.BanASNId > 0 {
		const queryUpdate = `
			UPDATE ban_asn 
			SET as_num = $2, origin = $3, source_id = $4, target_id = $5, reason = $6,
				valid_until = $7, updated_on = $8, reason_text = $9, is_enabled = $10, deleted = $11, 
				unban_reason_text = $12, appeal_state = $13
			WHERE ban_asn_id = $1`

		return errs.DBErr(s.
			Exec(ctx, queryUpdate, banASN.BanASNId, banASN.ASNum, banASN.Origin, banASN.SourceID.Int64(),
				banASN.TargetID.Int64(), banASN.Reason, banASN.ValidUntil, banASN.UpdatedOn, banASN.ReasonText, banASN.IsEnabled,
				banASN.Deleted, banASN.UnbanReasonText, banASN.AppealState))
	}

	const queryInsert = `
		INSERT INTO ban_asn (as_num, origin, source_id, target_id, reason, valid_until, updated_on, created_on, 
		                     reason_text, is_enabled, deleted, unban_reason_text, appeal_state)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING ban_asn_id`

	return errs.DBErr(s.
		QueryRow(ctx, queryInsert, banASN.ASNum, banASN.Origin, banASN.SourceID.Int64(), banASN.TargetID.Int64(),
			banASN.Reason, banASN.ValidUntil, banASN.UpdatedOn, banASN.CreatedOn, banASN.ReasonText, banASN.IsEnabled,
			banASN.Deleted, banASN.UnbanReasonText, banASN.AppealState).
		Scan(&banASN.BanASNId))
}

func (s Stores) DropBanASN(ctx context.Context, banASN *model.BanASN) error {
	banASN.Deleted = true

	return s.SaveBanASN(ctx, banASN)
}

func (s Stores) GetSteamIDsAtIP(ctx context.Context, ipNet *net.IPNet) (steamid.Collection, error) {
	const query = `
		SELECT DISTINCT c.steam_id
		FROM person_connections c
		WHERE ip_addr::inet <<= inet '%s';`

	if ipNet == nil {
		return nil, errs.ErrInvalidCIDR
	}

	rows, errQuery := s.Query(ctx, fmt.Sprintf(query, ipNet.String()))
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	var ids steamid.Collection

	for rows.Next() {
		var sid64 int64
		if errScan := rows.Scan(&sid64); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		ids = append(ids, steamid.New(sid64))
	}

	return ids, nil
}

func (s Stores) GetCIDRBlockSources(ctx context.Context) ([]model.CIDRBlockSource, error) {
	blocks := make([]model.CIDRBlockSource, 0)

	rows, errRows := s.QueryBuilder(ctx, s.
		Builder().
		Select("cidr_block_source_id", "name", "url", "enabled", "created_on", "updated_on").
		From("cidr_block_source"))
	if errRows != nil {
		if errors.Is(errRows, errs.ErrNoResult) {
			return blocks, nil
		}

		return nil, errs.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var block model.CIDRBlockSource
		if errScan := rows.Scan(&block.CIDRBlockSourceID, &block.Name, &block.URL, &block.Enabled, &block.CreatedOn, &block.UpdatedOn); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}

func (s Stores) GetCIDRBlockSource(ctx context.Context, sourceID int, block *model.CIDRBlockSource) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("cidr_block_source_id", "name", "url", "enabled", "created_on", "updated_on").
		From("cidr_block_source").
		Where(sq.Eq{"cidr_block_source_id": sourceID}))
	if errRow != nil {
		return errs.DBErr(errRow)
	}

	if errScan := row.Scan(&block.CIDRBlockSourceID, &block.Name, &block.URL, &block.Enabled, &block.CreatedOn, &block.UpdatedOn); errScan != nil {
		return errs.DBErr(errScan)
	}

	return nil
}

func (s Stores) SaveCIDRBlockSources(ctx context.Context, block *model.CIDRBlockSource) error {
	now := time.Now()

	block.UpdatedOn = now

	if block.CIDRBlockSourceID > 0 {
		return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
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

	return errs.DBErr(s.ExecInsertBuilderWithReturnValue(ctx, s.
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

func (s Stores) DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error {
	return errs.DBErr(s.ExecDeleteBuilder(ctx, s.
		Builder().
		Delete("cidr_block_source").
		Where(sq.Eq{"cidr_block_source_id": blockSourceID})))
}

func (s Stores) GetCIDRBlockWhitelists(ctx context.Context) ([]model.CIDRBlockWhitelist, error) {
	whitelists := make([]model.CIDRBlockWhitelist, 0)

	rows, errRows := s.QueryBuilder(ctx, s.
		Builder().
		Select("cidr_block_whitelist_id", "address", "created_on", "updated_on").
		From("cidr_block_whitelist"))
	if errRows != nil {
		if errors.Is(errRows, errs.ErrNoResult) {
			return whitelists, nil
		}

		return nil, errs.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var whitelist model.CIDRBlockWhitelist
		if errScan := rows.Scan(&whitelist.CIDRBlockWhitelistID, &whitelist.Address, &whitelist.CreatedOn, &whitelist.UpdatedOn); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		whitelists = append(whitelists, whitelist)
	}

	return whitelists, nil
}

func (s Stores) GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *model.CIDRBlockWhitelist) error {
	rows, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("cidr_block_whitelist_id", "address", "created_on", "updated_on").
		From("cidr_block_whitelist").
		Where(sq.Eq{"cidr_block_whitelist_id": whitelistID}))
	if errRow != nil {
		return errs.DBErr(errRow)
	}

	if errScan := rows.Scan(&whitelist.CIDRBlockWhitelistID, &whitelist.Address, &whitelist.CreatedOn, &whitelist.UpdatedOn); errScan != nil {
		return errs.DBErr(errScan)
	}

	return nil
}

func (s Stores) SaveCIDRBlockWhitelist(ctx context.Context, whitelist *model.CIDRBlockWhitelist) error {
	now := time.Now()

	whitelist.UpdatedOn = now

	if whitelist.CIDRBlockWhitelistID > 0 {
		return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
			Builder().
			Update("cidr_block_whitelist").
			SetMap(map[string]interface{}{
				"address":    whitelist.Address.String(),
				"updated_on": whitelist.UpdatedOn,
			})))
	}

	whitelist.CreatedOn = now

	return errs.DBErr(s.ExecInsertBuilderWithReturnValue(ctx, s.
		Builder().
		Insert("cidr_block_whitelist").
		SetMap(map[string]interface{}{
			"address":    whitelist.Address.String(),
			"created_on": whitelist.CreatedOn,
			"updated_on": whitelist.UpdatedOn,
		}).
		Suffix("RETURNING cidr_block_whitelist_id"), &whitelist.CIDRBlockWhitelistID))
}

func (s Stores) DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error {
	return errs.DBErr(s.ExecDeleteBuilder(ctx, s.
		Builder().
		Delete("cidr_block_whitelist").
		Where(sq.Eq{"cidr_block_whitelist_id": whitelistID})))
}
