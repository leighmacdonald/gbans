package store

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

// GetBanNet returns the BanNet matching intersecting the supplied ip.
//
// Note that this function does not currently limit results returned. This may change in the future, do not
// rely on this functionality.
func (database *pgStore) GetBanNet(ctx context.Context, ip net.IP) ([]model.BanNet, error) {
	const query = `
		SELECT net_id, cidr, source, created_on, updated_on, reason, reason_text, valid_until 
		FROM ban_net
		WHERE $1 <<= cidr`
	var nets []model.BanNet
	rows, errQuery := database.conn.Query(ctx, query, ip.String())
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var banNet model.BanNet
		if errScan := rows.Scan(&banNet.NetID, &banNet.CIDR, &banNet.Source,
			&banNet.CreatedOn, &banNet.UpdatedOn, &banNet.Reason, &banNet.ReasonText,
			&banNet.ValidUntil); errScan != nil {
			return nil, Err(errScan)
		}
		nets = append(nets, banNet)
	}
	return nets, nil
}

func (database *pgStore) updateBanNet(ctx context.Context, banNet *model.BanNet) error {
	query, args, errQueryArgs := sb.Update("ban_net").
		Set("cidr", banNet.CIDR).
		Set("source", banNet.Source).
		Set("created_on", banNet.CreatedOn).
		Set("updated_on", banNet.UpdatedOn).
		Set("reason", banNet.Reason).
		Set("reason_text", banNet.ReasonText).
		Set("valid_until_id", banNet.ValidUntil).
		Where(sq.Eq{"net_id": banNet.NetID}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	_, errExec := database.conn.Exec(ctx, query, args...)
	return Err(errExec)
}

func (database *pgStore) insertBanNet(ctx context.Context, banNet *model.BanNet) error {
	query, args, errQueryArgs := sb.Insert("ban_net").
		Columns("cidr", "source", "created_on", "updated_on", "reason", "reason_text", "valid_until").
		Values(banNet.CIDR, banNet.Source, banNet.CreatedOn, banNet.UpdatedOn, banNet.Reason, banNet.ReasonText, banNet.ValidUntil).
		Suffix("RETURNING net_id").
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	return Err(database.conn.QueryRow(ctx, query, args...).Scan(&banNet.NetID))
}

func (database *pgStore) SaveBanNet(ctx context.Context, banNet *model.BanNet) error {
	if banNet.NetID > 0 {
		return database.updateBanNet(ctx, banNet)
	}
	return database.insertBanNet(ctx, banNet)
}

func (database *pgStore) DropBanNet(ctx context.Context, banNet *model.BanNet) error {
	query, args, errQueryArgs := sb.Delete("ban_net").Where(sq.Eq{"net_id": banNet.NetID}).ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errExec := database.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	banNet.NetID = 0
	return nil
}

func (database *pgStore) GetExpiredNetBans(ctx context.Context) ([]model.BanNet, error) {
	const query = `
		SELECT net_id, cidr, source, created_on, updated_on, reason, reason_text, valid_until
		FROM ban_net
		WHERE valid_until < $1`
	var bans []model.BanNet
	rows, errQuery := database.Query(ctx, query, config.Now())
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var banNet model.BanNet
		if errScan := rows.Scan(&banNet.NetID, &banNet.CIDR, &banNet.Source, &banNet.CreatedOn, &banNet.UpdatedOn,
			&banNet.Reason, &banNet.ReasonText, &banNet.ValidUntil); errScan != nil {
			return nil, Err(errScan)
		}
		bans = append(bans, banNet)
	}
	return bans, nil
}

func (database *pgStore) GetExpiredASNBans(ctx context.Context) ([]model.BanASN, error) {
	const query = `
		SELECT ban_asn_id, as_num, origin, author_id, target_id, reason, reason_text, valid_until, created_on, updated_on
		FROM ban_asn
		WHERE valid_until < $1`
	var bans []model.BanASN
	rows, errQuery := database.conn.Query(ctx, query, config.Now())
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var banASN model.BanASN
		if errScan := rows.Scan(&banASN.BanASNId, &banASN.ASNum, &banASN.Origin, &banASN.AuthorID, &banASN.TargetID,
			&banASN.Reason, &banASN.ReasonText, &banASN.ValidUntil, &banASN.CreatedOn, &banASN.UpdatedOn); errScan != nil {
			return nil, errScan
		}
		bans = append(bans, banASN)
	}
	return bans, nil
}

func (database *pgStore) GetASNRecordsByNum(ctx context.Context, asNum int64) (ip2location.ASNRecords, error) {
	const query = `
		SELECT ip_from, ip_to, cidr, as_num, as_name 
		FROM net_asn
		WHERE as_num = $1`
	rows, errQuery := database.conn.Query(ctx, query, asNum)
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

func (database *pgStore) GetASNRecordByIP(ctx context.Context, ip net.IP, asnRecord *ip2location.ASNRecord) error {
	const query = `
		SELECT ip_from, ip_to, cidr, as_num, as_name 
		FROM net_asn
		WHERE $1::inet <@ ip_range
		LIMIT 1`
	if errQuery := database.conn.QueryRow(ctx, query, ip.String()).
		Scan(&asnRecord.IPFrom, &asnRecord.IPTo, &asnRecord.CIDR, &asnRecord.ASNum, &asnRecord.ASName); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) GetLocationRecord(ctx context.Context, ip net.IP, r *ip2location.LocationRecord) error {
	const query = `
		SELECT ip_from, ip_to, country_code, country_name, region_name, city_name, ST_Y(location), ST_X(location) 
		FROM net_location 
		WHERE $1::inet <@ ip_range`
	if errQuery := database.conn.QueryRow(ctx, query, ip.String()).
		Scan(&r.IPFrom, &r.IPTo, &r.CountryCode, &r.CountryName, &r.RegionName, &r.CityName, &r.LatLong.Latitude, &r.LatLong.Longitude); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) GetProxyRecord(ctx context.Context, ip net.IP, proxyRecord *ip2location.ProxyRecord) error {
	const query = `
		SELECT ip_from, ip_to, proxy_type, country_code, country_name, region_name, 
       		city_name, isp, domain_used, usage_type, as_num, as_name, last_seen, threat 
		FROM net_proxy 
		WHERE $1::inet <@ ip_range`
	if errQuery := database.conn.QueryRow(ctx, query, ip.String()).
		Scan(&proxyRecord.IPFrom, &proxyRecord.IPTo, &proxyRecord.ProxyType, &proxyRecord.CountryCode, &proxyRecord.CountryName, &proxyRecord.RegionName, &proxyRecord.CityName, &proxyRecord.ISP,
			&proxyRecord.Domain, &proxyRecord.UsageType, &proxyRecord.ASN, &proxyRecord.AS, &proxyRecord.LastSeen, &proxyRecord.Threat); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) loadASN(ctx context.Context, records []ip2location.ASNRecord) error {
	t0 := config.Now()
	if errTruncate := database.truncateTable(ctx, tableNetASN); errTruncate != nil {
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
				batchResults := database.conn.SendBatch(c, &batch)
				if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
					cancel()
					return errCloseBatch
				}
				cancel()
				batch = pgx.Batch{}
				log.Debugf("ASN Progress: %d/%d (%.0f%%)", recordIdx, len(records)-1, float64(recordIdx)/float64(len(records)-1)*100)
			}
		}
	}
	log.Debugf("Loaded %d ASN4 records in %s", len(records), time.Since(t0).String())
	return nil
}

func (database *pgStore) loadLocation(ctx context.Context, records []ip2location.LocationRecord, _ bool) error {
	t0 := config.Now()
	if errTruncate := database.truncateTable(ctx, tableNetLocation); errTruncate != nil {
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
				batchResults := database.conn.SendBatch(c, &batch)
				if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
					cancel()
					return errCloseBatch
				}
				cancel()
				batch = pgx.Batch{}
				log.Debugf("Location4 Progress: %d/%d (%.0f%%)", recordIdx, len(records)-1, float64(recordIdx)/float64(len(records)-1)*100)
			}
		}
	}
	log.Debugf("Loaded %d Location4 records in %s", len(records), time.Since(t0).String())
	return nil
}

func (database *pgStore) loadProxies(ctx context.Context, records []ip2location.ProxyRecord, _ bool) error {
	t0 := config.Now()
	if errTruncate := database.truncateTable(ctx, tableNetProxy); errTruncate != nil {
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
				batchResults := database.conn.SendBatch(c, &batch)
				if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
					cancel()
					return errCloseBatch
				}
				cancel()
				batch = pgx.Batch{}
				log.Debugf("Proxy Progress: %d/%d (%.0f%%)", recordIdx, len(records)-1, float64(recordIdx)/float64(len(records)-1)*100)
			}
		}
	}
	log.Debugf("Loaded %d Proxy records in %s", len(records), time.Since(t0).String())
	return nil
}

// InsertBlockListData will load the provided datasets into the database
//
// Note that this can take a while on slower machines. For reference, it takes
// about ~90s with a local database on a Ryzen 3900X/PCIe4 NVMe SSD.
func (database *pgStore) InsertBlockListData(ctx context.Context, blockListData *ip2location.BlockListData) error {
	if len(blockListData.Proxies) > 0 {
		if errProxies := database.loadProxies(ctx, blockListData.Proxies, false); errProxies != nil {
			return errProxies
		}
	}
	if len(blockListData.Locations4) > 0 {
		if errLocation := database.loadLocation(ctx, blockListData.Locations4, false); errLocation != nil {
			return errLocation
		}
	}
	if len(blockListData.ASN4) > 0 {
		if errASN := database.loadASN(ctx, blockListData.ASN4); errASN != nil {
			return errASN
		}
	}
	return nil
}

func (database *pgStore) GetBanASN(ctx context.Context, asNum int64, banASN *model.BanASN) error {
	const query = `
		SELECT ban_asn_id, as_num, origin, author_id, target_id, reason, reason_text, valid_until, created_on, updated_on 
		FROM ban_asn 
		WHERE as_num = $1`
	if errQuery := database.conn.QueryRow(ctx, query, asNum).Scan(&banASN.BanASNId, &banASN.ASNum, &banASN.Origin, &banASN.AuthorID,
		&banASN.TargetID, &banASN.Reason, &banASN.ReasonText, &banASN.ValidUntil, &banASN.CreatedOn, &banASN.UpdatedOn); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) SaveBanASN(ctx context.Context, banASN *model.BanASN) error {
	banASN.UpdatedOn = config.Now()
	if banASN.BanASNId > 0 {
		const queryUpdate = `
			UPDATE ban_asn 
			SET as_num = $2, origin = $3, author_id = $4, target_id = $5, reason = $6,
				valid_until = $7, updated_on = $8, reason_text = $9
			WHERE ban_asn_id = $1`

		_, errUpdate := database.conn.Exec(ctx, queryUpdate, banASN.BanASNId, banASN.ASNum, banASN.Origin, banASN.AuthorID, banASN.TargetID,
			banASN.Reason, banASN.ValidUntil, banASN.UpdatedOn, banASN.ReasonText)
		return Err(errUpdate)

	}
	const queryInsert = `
		INSERT INTO ban_asn (as_num, origin, author_id, target_id, reason, valid_until, updated_on, created_on, reason_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ban_asn_id`
	errInsert := database.conn.QueryRow(ctx, queryInsert, banASN.ASNum, banASN.Origin, banASN.AuthorID, banASN.TargetID,
		banASN.Reason, banASN.ValidUntil, banASN.UpdatedOn, banASN.CreatedOn, banASN.ReasonText).Scan(&banASN.BanASNId)
	return Err(errInsert)
}

func (database *pgStore) DropBanASN(ctx context.Context, banASN *model.BanASN) error {
	const query = `DELETE FROM ban_asn WHERE ban_asn_id = $1`
	_, errExec := database.conn.Exec(ctx, query, banASN.BanASNId)
	return Err(errExec)
}

func (database *pgStore) GetSteamIDsAtIP(ctx context.Context, ipNet *net.IPNet) (steamid.Collection, error) {
	const query = `
		SELECT DISTINCT source_id
		FROM server_log
		WHERE event_type = 1004 AND (meta_data->>'address')::inet <<= inet '%s';`
	if ipNet == nil {
		return nil, errors.New("Invalid address")
	}
	rows, errQuery := database.conn.Query(ctx, fmt.Sprintf(query, ipNet.String()))
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
