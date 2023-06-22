package store

import (
	"context"
	"fmt"
	"net"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// GetBanNetByAddress returns the BanCIDR matching intersecting the supplied ip.
//
// Note that this function does not currently limit results returned. This may change in the future, do not
// rely on this functionality.
func GetBanNetByAddress(ctx context.Context, ip net.IP) ([]BanCIDR, error) {
	const query = `
		SELECT net_id, cidr, origin, created_on, updated_on, reason, reason_text, valid_until, deleted, 
		       note, unban_reason_text, is_enabled, target_id, source_id, appeal_state
		FROM ban_net
		WHERE $1 <<= cidr AND deleted = false AND is_enabled = true`
	var nets []BanCIDR
	rows, errQuery := Query(ctx, query, ip.String())
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var banNet BanCIDR
		if errScan := rows.Scan(&banNet.NetID, &banNet.CIDR, &banNet.Origin,
			&banNet.CreatedOn, &banNet.UpdatedOn, &banNet.Reason, &banNet.ReasonText,
			&banNet.ValidUntil, &banNet.Deleted, &banNet.Note, &banNet.UnbanReasonText,
			&banNet.IsEnabled, &banNet.TargetID, &banNet.SourceID, &banNet.AppealState); errScan != nil {
			return nil, Err(errScan)
		}
		nets = append(nets, banNet)
	}
	return nets, nil
}

func GetBanNetById(ctx context.Context, netID int64, banNet *BanCIDR) error {
	const query = `
		SELECT net_id, cidr, origin, created_on, updated_on, reason, reason_text, valid_until, deleted, 
		       note, unban_reason_text, is_enabled, target_id, source_id, appeal_state
		FROM ban_net
		WHERE deleted = false AND net_id = $1`
	return Err(QueryRow(ctx, query, netID).Scan(&banNet.NetID, &banNet.CIDR, &banNet.Origin,
		&banNet.CreatedOn, &banNet.UpdatedOn, &banNet.Reason, &banNet.ReasonText,
		&banNet.ValidUntil, &banNet.Deleted, &banNet.Note, &banNet.UnbanReasonText,
		&banNet.IsEnabled, &banNet.TargetID, &banNet.SourceID, &banNet.AppealState))
}

// GetBansNet returns the BanCIDR matching intersecting the supplied ip.
func GetBansNet(ctx context.Context) ([]BanCIDR, error) {
	const query = `
		SELECT net_id, cidr, origin, created_on, updated_on, reason, reason_text, valid_until, deleted, 
		       note, unban_reason_text, is_enabled, target_id, source_id, appeal_state 
		FROM ban_net
		WHERE deleted = false`
	var nets []BanCIDR
	rows, errQuery := Query(ctx, query)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var banNet BanCIDR
		if errScan := rows.Scan(&banNet.NetID, &banNet.CIDR, &banNet.Origin,
			&banNet.CreatedOn, &banNet.UpdatedOn, &banNet.Reason, &banNet.ReasonText,
			&banNet.ValidUntil, &banNet.Deleted, &banNet.Note, &banNet.UnbanReasonText,
			&banNet.IsEnabled, &banNet.TargetID, &banNet.SourceID, &banNet.AppealState); errScan != nil {
			return nil, Err(errScan)
		}
		nets = append(nets, banNet)
	}
	return nets, nil
}

func updateBanNet(ctx context.Context, banNet *BanCIDR) error {
	banNet.UpdatedOn = config.Now()
	query, args, errQueryArgs := sb.
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
		Set("target_id", banNet.TargetID).
		Set("source_id", banNet.SourceID).
		Set("appeal_state", banNet.AppealState).
		Where(sq.Eq{"net_id": banNet.NetID}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	return Err(Exec(ctx, query, args...))
}

func insertBanNet(ctx context.Context, banNet *BanCIDR) error {
	query, args, errQueryArgs := sb.
		Insert("ban_net").
		Columns("cidr", "origin", "created_on", "updated_on", "reason", "reason_text", "valid_until",
			"deleted", "note", "unban_reason_text", "is_enabled", "target_id", "source_id", "appeal_state").
		Values(banNet.CIDR, banNet.Origin, banNet.CreatedOn, banNet.UpdatedOn, banNet.Reason, banNet.ReasonText,
			banNet.ValidUntil, banNet.Deleted, banNet.Note, banNet.UnbanReasonText, banNet.IsEnabled,
			banNet.TargetID, banNet.SourceID, banNet.AppealState).
		Suffix("RETURNING net_id").
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	return Err(QueryRow(ctx, query, args...).Scan(&banNet.NetID))
}

func SaveBanNet(ctx context.Context, banNet *BanCIDR) error {
	if banNet.NetID > 0 {
		return updateBanNet(ctx, banNet)
	}
	return insertBanNet(ctx, banNet)
}

func DropBanNet(ctx context.Context, banNet *BanCIDR) error {
	query, args, errQueryArgs := sb.Delete("ban_net").Where(sq.Eq{"net_id": banNet.NetID}).ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errExec := Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	banNet.NetID = 0
	return nil
}

func GetExpiredNetBans(ctx context.Context) ([]BanCIDR, error) {
	const query = `
		SELECT net_id, cidr, origin, created_on, updated_on, reason_text, valid_until, deleted, note, 
		       unban_reason_text, is_enabled, target_id, source_id, reason, appeal_state
		FROM ban_net
		WHERE valid_until < $1`
	var bans []BanCIDR
	rows, errQuery := Query(ctx, query, config.Now())
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var banNet BanCIDR
		if errScan := rows.Scan(&banNet.NetID, &banNet.CIDR, &banNet.Origin, &banNet.CreatedOn,
			&banNet.UpdatedOn, &banNet.ReasonText, &banNet.ValidUntil, &banNet.Deleted, &banNet.Note,
			&banNet.UnbanReasonText, &banNet.IsEnabled, &banNet.TargetID, &banNet.SourceID,
			&banNet.Reason, &banNet.AppealState); errScan != nil {
			return nil, Err(errScan)
		}
		bans = append(bans, banNet)
	}
	return bans, nil
}

func GetExpiredASNBans(ctx context.Context) ([]BanASN, error) {
	const query = `
		SELECT ban_asn_id, as_num, origin, source_id, target_id, reason_text, valid_until, created_on, updated_on, 
		       deleted, reason, is_enabled, unban_reason_text, appeal_state
		FROM ban_asn
		WHERE valid_until < $1 AND deleted = false`
	var bans []BanASN
	rows, errQuery := conn.Query(ctx, query, config.Now())
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var banASN BanASN
		if errScan := rows.Scan(&banASN.BanASNId, &banASN.ASNum, &banASN.Origin, &banASN.SourceID, &banASN.TargetID,
			&banASN.ReasonText, &banASN.ValidUntil, &banASN.CreatedOn, &banASN.UpdatedOn, &banASN.Deleted,
			&banASN.Reason, &banASN.IsEnabled, &banASN.UnbanReasonText, &banASN.AppealState); errScan != nil {
			return nil, errScan
		}
		bans = append(bans, banASN)
	}
	return bans, nil
}

func GetASNRecordsByNum(ctx context.Context, asNum int64) (ip2location.ASNRecords, error) {
	const query = `
		SELECT ip_from, ip_to, cidr, as_num, as_name 
		FROM net_asn
		WHERE as_num = $1`
	rows, errQuery := conn.Query(ctx, query, asNum)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	var records ip2location.ASNRecords
	for rows.Next() {
		var asnRecord ip2location.ASNRecord
		if errScan := rows.Scan(&asnRecord.IPFrom, &asnRecord.IPTo, &asnRecord.CIDR, &asnRecord.ASNum, &asnRecord.ASName); errScan != nil {
			return nil, Err(errScan)
		}
		records = append(records, asnRecord)
	}
	return records, nil
}

func GetASNRecordByIP(ctx context.Context, ip net.IP, asnRecord *ip2location.ASNRecord) error {
	const query = `
		SELECT ip_from, ip_to, cidr, as_num, as_name 
		FROM net_asn
		WHERE $1::inet <@ ip_range
		LIMIT 1`
	if errQuery := conn.QueryRow(ctx, query, ip.String()).
		Scan(&asnRecord.IPFrom, &asnRecord.IPTo, &asnRecord.CIDR, &asnRecord.ASNum, &asnRecord.ASName); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func GetLocationRecord(ctx context.Context, ip net.IP, r *ip2location.LocationRecord) error {
	const query = `
		SELECT ip_from, ip_to, country_code, country_name, region_name, city_name, ST_Y(location), ST_X(location) 
		FROM net_location 
		WHERE ip_range @> $1::inet`
	if errQuery := QueryRow(ctx, query, ip.String()).
		Scan(&r.IPFrom, &r.IPTo, &r.CountryCode, &r.CountryName, &r.RegionName, &r.CityName, &r.LatLong.Latitude, &r.LatLong.Longitude); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func GetProxyRecord(ctx context.Context, ip net.IP, proxyRecord *ip2location.ProxyRecord) error {
	const query = `
		SELECT ip_from, ip_to, proxy_type, country_code, country_name, region_name, 
       		city_name, isp, domain_used, usage_type, as_num, as_name, last_seen, threat 
		FROM net_proxy 
		WHERE $1::inet <@ ip_range`
	if errQuery := QueryRow(ctx, query, ip.String()).
		Scan(&proxyRecord.IPFrom, &proxyRecord.IPTo, &proxyRecord.ProxyType, &proxyRecord.CountryCode, &proxyRecord.CountryName, &proxyRecord.RegionName, &proxyRecord.CityName, &proxyRecord.ISP,
			&proxyRecord.Domain, &proxyRecord.UsageType, &proxyRecord.ASN, &proxyRecord.AS, &proxyRecord.LastSeen, &proxyRecord.Threat); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func loadASN(ctx context.Context, records []ip2location.ASNRecord) error {
	t0 := config.Now()
	if errTruncate := truncateTable(ctx, tableNetASN); errTruncate != nil {
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
				batchResults := conn.SendBatch(c, &batch)
				if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
					cancel()
					return errCloseBatch
				}
				cancel()
				batch = pgx.Batch{}
				logger.Info(fmt.Sprintf("ASN Progress: %d/%d (%.0f%%)",
					recordIdx, len(records)-1, float64(recordIdx)/float64(len(records)-1)*100))
			}
		}
	}
	logger.Info("Loaded ASN4 records",
		zap.Int("count", len(records)), zap.Duration("duration", time.Since(t0)))
	return nil
}

func loadLocation(ctx context.Context, records []ip2location.LocationRecord, _ bool) error {
	t0 := config.Now()
	if errTruncate := truncateTable(ctx, tableNetLocation); errTruncate != nil {
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
				batchResults := conn.SendBatch(c, &batch)
				if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
					cancel()
					return errCloseBatch
				}
				cancel()
				batch = pgx.Batch{}
				logger.Debug(fmt.Sprintf("Location4 Progress: %d/%d (%.0f%%)",
					recordIdx, len(records)-1, float64(recordIdx)/float64(len(records)-1)*100))
			}
		}
	}
	logger.Debug("Loaded Location4 records",
		zap.Int("count", len(records)), zap.Duration("duration", time.Since(t0)))
	return nil
}

func loadProxies(ctx context.Context, records []ip2location.ProxyRecord, _ bool) error {
	t0 := config.Now()
	if errTruncate := truncateTable(ctx, tableNetProxy); errTruncate != nil {
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
				c, cancel := context.WithTimeout(ctx, time.Second*10)
				batchResults := conn.SendBatch(c, &batch)
				if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
					cancel()
					return errCloseBatch
				}
				cancel()
				batch = pgx.Batch{}
				logger.Debug(fmt.Sprintf("Proxy Progress: %d/%d (%.0f%%)",
					recordIdx, len(records)-1, float64(recordIdx)/float64(len(records)-1)*100))
			}
		}
	}
	logger.Debug("Loaded Proxy records",
		zap.Int("count", len(records)), zap.Duration("duration", time.Since(t0)))
	return nil
}

// InsertBlockListData will load the provided datasets into the database
//
// Note that this can take a while on slower machines. For reference, it takes
// about ~90s with a local database on a Ryzen 3900X/PCIe4 NVMe SSD.
func InsertBlockListData(ctx context.Context, blockListData *ip2location.BlockListData) error {
	if len(blockListData.Proxies) > 0 {
		if errProxies := loadProxies(ctx, blockListData.Proxies, false); errProxies != nil {
			return errProxies
		}
	}
	if len(blockListData.Locations4) > 0 {
		if errLocation := loadLocation(ctx, blockListData.Locations4, false); errLocation != nil {
			return errLocation
		}
	}
	if len(blockListData.ASN4) > 0 {
		if errASN := loadASN(ctx, blockListData.ASN4); errASN != nil {
			return errASN
		}
	}
	return nil
}

func GetBanASN(ctx context.Context, asNum int64, banASN *BanASN) error {
	const query = `
		SELECT ban_asn_id, as_num, origin, source_id, target_id, reason_text, valid_until, created_on, updated_on, 
		       deleted, reason, is_enabled, unban_reason_text, appeal_state
		FROM ban_asn 
		WHERE deleted = false AND as_num = $1`
	if errQuery := QueryRow(ctx, query, asNum).Scan(&banASN.BanASNId, &banASN.ASNum, &banASN.Origin,
		&banASN.SourceID, &banASN.TargetID, &banASN.ReasonText, &banASN.ValidUntil, &banASN.CreatedOn,
		&banASN.UpdatedOn, &banASN.Deleted, &banASN.Reason, &banASN.IsEnabled, &banASN.UnbanReasonText,
		&banASN.AppealState); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func GetBansASN(ctx context.Context) ([]BanASN, error) {
	const query = `
		SELECT ban_asn_id, as_num, origin, source_id, target_id, reason_text, valid_until, created_on, updated_on, 
		       deleted, reason, is_enabled, unban_reason_text, appeal_state
		FROM ban_asn 
		WHERE deleted = false`
	rows, errRows := Query(ctx, query)
	if errRows != nil {
		return nil, Err(errRows)
	}
	defer rows.Close()
	var records []BanASN
	for rows.Next() {
		var r BanASN
		if errQuery := rows.Scan(&r.BanASNId, &r.ASNum, &r.Origin,
			&r.SourceID, &r.TargetID, &r.ReasonText, &r.ValidUntil, &r.CreatedOn,
			&r.UpdatedOn, &r.Deleted, &r.Reason, &r.IsEnabled, &r.UnbanReasonText,
			&r.AppealState); errQuery != nil {
			return nil, Err(errQuery)
		}
		records = append(records, r)
	}
	return records, nil
}

func SaveBanASN(ctx context.Context, banASN *BanASN) error {
	banASN.UpdatedOn = config.Now()
	if banASN.BanASNId > 0 {
		const queryUpdate = `
			UPDATE ban_asn 
			SET as_num = $2, origin = $3, source_id = $4, target_id = $5, reason = $6,
				valid_until = $7, updated_on = $8, reason_text = $9, is_enabled = $10, deleted = $11, 
				unban_reason_text = $12, appeal_state = $13
			WHERE ban_asn_id = $1`
		return Err(Exec(ctx, queryUpdate, banASN.BanASNId, banASN.ASNum, banASN.Origin, banASN.SourceID,
			banASN.TargetID, banASN.Reason, banASN.ValidUntil, banASN.UpdatedOn, banASN.ReasonText, banASN.IsEnabled,
			banASN.Deleted, banASN.UnbanReasonText, banASN.AppealState))
	}
	const queryInsert = `
		INSERT INTO ban_asn (as_num, origin, source_id, target_id, reason, valid_until, updated_on, created_on, 
		                     reason_text, is_enabled, deleted, unban_reason_text, appeal_state)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING ban_asn_id`
	return Err(QueryRow(ctx, queryInsert, banASN.ASNum, banASN.Origin, banASN.SourceID, banASN.TargetID,
		banASN.Reason, banASN.ValidUntil, banASN.UpdatedOn, banASN.CreatedOn, banASN.ReasonText, banASN.IsEnabled,
		banASN.Deleted, banASN.UnbanReasonText, banASN.AppealState).Scan(&banASN.BanASNId))
}

func DropBanASN(ctx context.Context, banASN *BanASN) error {
	banASN.Deleted = true
	return SaveBanASN(ctx, banASN)
}

func GetSteamIDsAtIP(ctx context.Context, ipNet *net.IPNet) (steamid.Collection, error) {
	const query = `
		SELECT DISTINCT source_id
		FROM server_log
		WHERE event_type = 1004 AND (meta_data->>'address')::inet <<= inet '%s';`
	if ipNet == nil {
		return nil, errors.New("Invalid address")
	}
	rows, errQuery := Query(ctx, fmt.Sprintf(query, ipNet.String()))
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	var ids steamid.Collection
	for rows.Next() {
		var sid64 steamid.SID64
		if errScan := rows.Scan(&sid64); errScan != nil {
			return nil, Err(errScan)
		}
		ids = append(ids, sid64)
	}
	return ids, nil
}
