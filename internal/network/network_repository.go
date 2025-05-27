package network

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type networkRepository struct {
	db database.Database
}

func NewNetworkRepository(db database.Database) domain.NetworkRepository {
	return networkRepository{db: db}
}

func (r networkRepository) QueryConnections(ctx context.Context, opts domain.ConnectionHistoryQuery) ([]domain.PersonConnection, int64, error) {
	var constraints sq.And

	if opts.Sid64 != "" {
		sid := steamid.New(opts.Sid64)
		if !sid.Valid() {
			return nil, 0, domain.ErrInvalidSID
		}

		constraints = append(constraints, sq.Eq{"steam_id": sid.Int64()})
	}

	if opts.Network != "" {
		constraints = append(constraints, sq.Expr("ip_addr <<= ?::ip4r", opts.Network))
	}

	selectBuilder := r.db.Builder().
		Select("distinct on (c.ip_addr) c.ip_addr", "c.person_connection_id", "c.persona_name",
			"c.steam_id", "c.created_on", "c.server_id", "s.short_name", "s.name").
		From("person_connections c").
		LeftJoin("server s USING(server_id)").
		Where(constraints)

	builder := r.db.
		Builder().
		Select("x.*").
		FromSelect(selectBuilder, "x")

	builder = opts.ApplySafeOrder(opts.ApplyLimitOffsetDefault(builder), map[string][]string{
		"x.": {"steam_id", "ip_addr", "persona_name", "created_on", "short_name", "name"},
	}, "created_on")

	var messages []domain.PersonConnection

	rows, errQuery := r.db.QueryBuilder(ctx, nil, builder.Where(constraints))
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

		if errScan := rows.Scan(
			&connHistory.IPAddr,
			&connHistory.PersonConnectionID,
			&connHistory.PersonaName,
			&steamID,
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

	count, errCount := r.db.GetCount(ctx, nil, r.db.
		Builder().
		Select("count(c.person_connection_id)").
		From("person_connections c").
		Where(constraints))

	if errCount != nil {
		return nil, 0, r.db.DBErr(errCount)
	}

	return messages, count, nil
}

func (r networkRepository) GetPersonIPHistory(ctx context.Context, sid64 steamid.SteamID, limit uint64) (domain.PersonConnections, error) {
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

	rows, errQuery := r.db.QueryBuilder(ctx, nil, builder)
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

func (r networkRepository) AddConnectionHistory(ctx context.Context, conn *domain.PersonConnection) error {
	const query = `
		INSERT INTO person_connections (steam_id, ip_addr, persona_name, created_on, server_id) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING person_connection_id`

	if errQuery := r.db.
		QueryRow(ctx, nil, query, conn.SteamID.Int64(), conn.IPAddr.String(), conn.PersonaName, conn.CreatedOn, conn.ServerID).
		Scan(&conn.PersonConnectionID); errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	return nil
}

func (r networkRepository) GetPlayerMostRecentIP(ctx context.Context, steamID steamid.SteamID) net.IP {
	row, errRow := r.db.QueryRowBuilder(ctx, nil, r.db.
		Builder().
		Select("c.ip_addr").
		From("person_connections c").
		Where(sq.Eq{"c.steam_id": steamID.Int64()}).
		OrderBy("c.created_on desc").
		Limit(1))
	if errRow != nil {
		if errors.Is(errRow, domain.ErrNoResult) {
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

func (r networkRepository) GetASNRecordsByNum(ctx context.Context, asNum int64) ([]domain.NetworkASN, error) {
	query := r.db.
		Builder().
		Select("cidr::text", "as_num", "as_name").
		From("net_asn").
		Where(sq.Eq{"as_num": asNum})

	rows, errQuery := r.db.QueryBuilder(ctx, nil, query)
	if errQuery != nil {
		return nil, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	var records []domain.NetworkASN

	for rows.Next() {
		var asnRecord domain.NetworkASN
		if errScan := rows.
			Scan(&asnRecord.CIDR, &asnRecord.ASNum, &asnRecord.ASName); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		records = append(records, asnRecord)
	}

	return records, nil
}

func (r networkRepository) GetASNRecordByIP(ctx context.Context, ipAddr netip.Addr) (domain.NetworkASN, error) {
	const query = `
		SELECT ip_range::text, as_num, as_name 
		FROM net_asn
		WHERE ip_range >>= $1 
		LIMIT 1`

	var asnRecord domain.NetworkASN

	if errQuery := r.db.
		QueryRow(ctx, nil, query, ipAddr.String()).
		Scan(&asnRecord.CIDR, &asnRecord.ASNum, &asnRecord.ASName); errQuery != nil {
		return asnRecord, r.db.DBErr(errQuery)
	}

	return asnRecord, nil
}

func (r networkRepository) GetLocationRecord(ctx context.Context, ipAddr netip.Addr) (domain.NetworkLocation, error) {
	const query = `
		SELECT ip_range::text, country_code, country_name, region_name, city_name, ST_Y(location), ST_X(location) 
		FROM net_location 
		WHERE ip_range >>= $1`

	var record domain.NetworkLocation

	if errQuery := r.db.QueryRow(ctx, nil, query, ipAddr.String()).
		Scan(&record.CIDR, &record.CountryCode, &record.CountryName, &record.RegionName,
			&record.CityName, &record.LatLong.Latitude, &record.LatLong.Longitude); errQuery != nil {
		return record, r.db.DBErr(errQuery)
	}

	return record, nil
}

func (r networkRepository) GetProxyRecord(ctx context.Context, ipAddr netip.Addr) (domain.NetworkProxy, error) {
	const query = `
		SELECT ip_range::text, proxy_type, country_code, country_name, region_name, 
       		city_name, isp, domain_used, usage_type, as_num, as_name, last_seen, threat 
		FROM net_proxy 
		WHERE ip_range >>= $1`

	var proxyRecord domain.NetworkProxy

	if errQuery := r.db.QueryRow(ctx, nil, query, ipAddr.String()).
		Scan(&proxyRecord.CIDR, &proxyRecord.ProxyType, &proxyRecord.CountryCode, &proxyRecord.CountryName, &proxyRecord.RegionName, &proxyRecord.CityName, &proxyRecord.ISP,
			&proxyRecord.Domain, &proxyRecord.UsageType, &proxyRecord.ASN, &proxyRecord.AS, &proxyRecord.LastSeen, &proxyRecord.Threat); errQuery != nil {
		return proxyRecord, r.db.DBErr(errQuery)
	}

	return proxyRecord, nil
}

func (r networkRepository) LoadASN(ctx context.Context, truncate bool, records []ip2location.ASNRecord) error {
	if truncate {
		slog.Debug("Truncating asn table")

		if errTruncate := r.db.TruncateTable(ctx, "net_asn"); errTruncate != nil {
			return errTruncate
		}
	}

	const query = `
		INSERT INTO net_asn (ip_range, cidr, as_num, as_name) 
		VALUES($1, $2, $3, $4)`

	batch := pgx.Batch{}

	for _, rec := range records {
		batch.Queue(query, fmt.Sprintf("%s-%s", rec.IPFrom, rec.IPTo), rec.CIDR, rec.ASNum, rec.ASName)
	}

	c, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	batchResults := r.db.SendBatch(c, nil, &batch)
	if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
		return errors.Join(errCloseBatch, domain.ErrCloseBatch)
	}

	return nil
}

func (r networkRepository) LoadLocation(ctx context.Context, truncate bool, records []ip2location.LocationRecord) error {
	if truncate {
		slog.Debug("Truncating location table")

		if errTruncate := r.db.TruncateTable(ctx, "net_location"); errTruncate != nil {
			return errTruncate
		}
	}

	const query = `
		INSERT INTO net_location (ip_range, country_code, country_name, region_name, city_name, location)
		VALUES($1::ip4r, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($7, $6), 4326) )`

	batch := pgx.Batch{}

	for _, rec := range records {
		batch.Queue(query, fmt.Sprintf("%s-%s", rec.IPFrom, rec.IPTo), rec.CountryCode, rec.CountryName, rec.RegionName, rec.CityName, rec.LatLong.Latitude, rec.LatLong.Longitude)
	}

	c, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	batchResults := r.db.SendBatch(c, nil, &batch)
	if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
		return errors.Join(errCloseBatch, domain.ErrCloseBatch)
	}

	return nil
}

func (r networkRepository) LoadProxies(ctx context.Context, truncate bool, records []ip2location.ProxyRecord) error {
	if truncate {
		slog.Debug("Truncating proxy table")

		if errTruncate := r.db.TruncateTable(ctx, "net_proxy"); errTruncate != nil {
			return errTruncate
		}
	}

	const query = `
		INSERT INTO net_proxy (ip_range, proxy_type, country_code, country_name, region_name, city_name, isp,
		                       domain_used, usage_type, as_num, as_name, last_seen, threat)
		VALUES(ip4r($1::ip4, $2::ip4), $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	batch := pgx.Batch{}

	for _, rec := range records {
		batch.Queue(query, rec.IPFrom.To4().String(), rec.IPTo.To4().String(), rec.ProxyType, rec.CountryCode, rec.CountryName, rec.RegionName, rec.CityName,
			rec.ISP, rec.Domain, rec.UsageType, rec.ASN, rec.AS, rec.LastSeen, rec.Threat)
	}

	c, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	batchResults := r.db.SendBatch(c, nil, &batch)
	if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
		return errors.Join(errCloseBatch, domain.ErrCloseBatch)
	}

	return nil
}
