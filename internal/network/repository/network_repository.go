package repository

import (
	"errors"
	"fmt"
	"net"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

// todo move to network
func (r *personRepository) QueryConnectionHistory(ctx context.Context, opts domain.ConnectionHistoryQueryFilter) ([]domain.PersonConnection, int64, error) {
	builder := r.db.
		Builder().
		Select("c.person_connection_id", "c.steam_id",
			"c.ip_addr", "c.persona_name", "c.created_on", "c.server_id", "r.short_name", "r.name").
		From("person_connections c").
		LeftJoin("server r USING(server_id)").
		GroupBy("c.person_connection_id, c.ip_addr, r.short_name", "r.name")

	var constraints sq.And

	if opts.SourceID != "" {
		sid, errSID := opts.SourceID.SID64(ctx)
		if errSID != nil {
			return nil, 0, errors.Join(steamid.ErrInvalidSID, domain.ErrSourceID)
		}

		constraints = append(constraints, sq.Eq{"c.steam_id": sid.Int64()})
	}

	builder = opts.ApplySafeOrder(opts.ApplyLimitOffsetDefault(builder), map[string][]string{
		"c.": {"person_connection_id", "steam_id", "ip_addr", "persona_name", "created_on"},
		"r.": {"short_name", "name"},
	}, "person_connection_id")

	var messages []domain.PersonConnection

	rows, errQuery := r.db.QueryBuilder(ctx, builder.Where(constraints))
	if errQuery != nil {
		return nil, 0, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			connHistory domain.PersonConnection
			steamID     int64
			serverID    *int
			shortName   *string
			name        *string
		)

		if errScan := rows.Scan(&connHistory.PersonConnectionID,
			&steamID,
			&connHistory.IPAddr,
			&connHistory.PersonaName,
			&connHistory.CreatedOn,
			&serverID, &shortName, &name); errScan != nil {
			return nil, 0, r.db.DBErr(errScan)
		}

		// Added later in dev, so can be legacy data w/o a server_id
		if serverID != nil && shortName != nil && name != nil {
			connHistory.ServerID = *serverID
			connHistory.ServerNameShort = *shortName
			connHistory.ServerName = *name
		}

		connHistory.SteamID = steamid.New(steamID)

		messages = append(messages, connHistory)
	}

	if messages == nil {
		return []domain.PersonConnection{}, 0, nil
	}

	count, errCount := r.db.GetCount(ctx, r.db.
		Builder().
		Select("count(c.person_connection_id)").
		From("person_connections c").
		Where(constraints))

	if errCount != nil {
		return nil, 0, r.db.DBErr(errCount)
	}

	return messages, count, nil
}

// todo move to network
func (r *personRepository) GetPersonIPHistory(ctx context.Context, sid64 steamid.SID64, limit uint64) (domain.PersonConnections, error) {
	builder := r.db.
		Builder().
		Select(
			"DISTINCT on (pn, pc.ip_addr) coalesce(pc.persona_name, pc.steam_id::text) as pn",
			"pc.person_connection_id",
			"pc.steam_id",
			"pc.ip_addr",
			"pc.created_on",
			"pc.server_id").
		From("person_connections pc").
		LeftJoin("net_location loc ON pc.ip_addr <@ loc.ip_range").
		// Join("LEFT JOIN net_proxy proxy ON pc.ip_addr <@ proxy.ip_range").
		OrderBy("1").
		Limit(limit)
	builder = builder.Where(sq.Eq{"pc.steam_id": sid64.Int64()})

	rows, errQuery := r.db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	var connections domain.PersonConnections

	for rows.Next() {
		var (
			conn    domain.PersonConnection
			steamID int64
		)

		if errScan := rows.Scan(&conn.PersonaName, &conn.PersonConnectionID, &steamID,
			&conn.IPAddr, &conn.CreatedOn, &conn.ServerID); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		conn.SteamID = steamid.New(steamID)

		connections = append(connections, conn)
	}

	return connections, nil
}

// todo move to network
func (r *personRepository) AddConnectionHistory(ctx context.Context, conn *domain.PersonConnection) error {
	const query = `
		INSERT INTO person_connections (steam_id, ip_addr, persona_name, created_on, server_id) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING person_connection_id`

	if errQuery := r.db.
		QueryRow(ctx, query, conn.SteamID.Int64(), conn.IPAddr, conn.PersonaName, conn.CreatedOn, conn.ServerID).
		Scan(&conn.PersonConnectionID); errQuery != nil {
		return r.db.DBErr(errQuery)
	}

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
