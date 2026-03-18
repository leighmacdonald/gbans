package network

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/network/ip2location"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct {
	database.Database

	persons person.Provider
}

func NewRepository(db database.Database, persons person.Provider) Repository {
	return Repository{Database: db, persons: persons}
}

func (r Repository) QueryConnections(ctx context.Context, opts ConnectionHistoryQuery) ([]PersonConnection, error) {
	var constraints sq.And

	if opts.SourceID != "" {
		sid := steamid.New(opts.SourceID)
		if !sid.Valid() {
			return nil, steamid.ErrInvalidSID
		}

		constraints = append(constraints, sq.Eq{"c.steam_id": sid.Int64()})
	}

	if opts.CIDR != "" {
		constraints = append(constraints, sq.Expr("c.ip_addr <<= ?::ip4r", opts.CIDR))
	}
	if len(opts.ServerID) > 0 {
		constraints = append(constraints, sq.Eq{"c.server_id": opts.ServerID})
	}
	if opts.ASNum > 0 {
		constraints = append(constraints, sq.Eq{"a.as_num": opts.ASNum})
	}
	if opts.ASName != "" {
		constraints = append(constraints, sq.Eq{"a.as_name": opts.ASName})
	}
	if opts.CountryCode != "" {
		constraints = append(constraints, sq.Eq{"l.country_code": strings.ToUpper(opts.CountryCode)})
	}
	if opts.CountryName != "" {
		constraints = append(constraints, sq.Eq{"l.country_name": opts.CountryName})
	}
	if opts.CityName != "" {
		constraints = append(constraints, sq.Eq{"l.city_name": opts.CityName})
	}

	selectBuilder := r.Builder().
		// distinct on (c.ip_addr) c.ip_addr
		Select(
			"c.ip_addr",
			"c.person_connection_id",
			"c.persona_name",
			"c.steam_id",
			"c.created_on",
			"c.server_id",
			"s.short_name",
			"s.name",
			"l.country_code",
			"l.country_name",
			"l.city_name",
			"ST_Y(l.location)", "ST_X(l.location)",
			"a.as_num",
			"a.as_name").
		From("person_connections c").
		LeftJoin("server s USING(server_id)").
		LeftJoin("net_location l ON (l.ip_range >>= c.ip_addr)").
		LeftJoin("net_asn a ON (a.ip_range >>= c.ip_addr)")

	selectBuilder = opts.ApplySafeOrder(opts.ApplyLimitOffsetDefault(selectBuilder), map[string][]string{
		"c.": {"steam_id", "ip_addr", "persona_name", "created_on", "short_name", "name", "server_id"},
		"s.": {"short_name", "name"},
		"l.": {"country_code", "country_name", "city_name"},
		"a.": {"as_num", "as_name"},
	}, "c.created_on")

	var messages []PersonConnection

	rows, errQuery := r.QueryBuilder(ctx, selectBuilder.Where(constraints))
	if errQuery != nil {
		return nil, database.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			connHistory PersonConnection
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
			&serverID, &shortName, &name,
			&connHistory.CountryCode, &connHistory.CountryName, &connHistory.CityName,
			&connHistory.LatLong.Latitude, &connHistory.LatLong.Longitude,
			&connHistory.ASNum, &connHistory.ASName); errScan != nil {
			return nil, database.DBErr(errScan)
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
		return []PersonConnection{}, nil
	}

	return messages, nil
}

func (r Repository) GetPersonIPHistory(ctx context.Context, sid64 steamid.SteamID, limit uint64) (PersonConnections, error) {
	builder := r.Builder().
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

	rows, errQuery := r.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, database.DBErr(errQuery)
	}

	defer rows.Close()

	var connections PersonConnections

	for rows.Next() {
		var (
			conn    PersonConnection
			steamID int64
		)

		if errScan := rows.Scan(&conn.PersonaName, &conn.PersonConnectionID, &steamID,
			&conn.IPAddr, &conn.CreatedOn, &conn.ServerID); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		conn.SteamID = steamid.New(steamID)

		connections = append(connections, conn)
	}

	return connections, nil
}

func (r Repository) AddConnectionHistory(ctx context.Context, conn *PersonConnection) error {
	const query = `
		INSERT INTO person_connections (steam_id, ip_addr, persona_name, created_on, server_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING person_connection_id`

	// Maybe ignore these and wait for connect call to create?

	if errPerson := r.persons.EnsurePerson(ctx, conn.SteamID); errPerson != nil {
		slog.Error("Failed to fetch connecting person", slog.String("steam_id", conn.SteamID.String()), slog.String("error", errPerson.Error()))

		return errPerson
	}

	if errQuery := r.
		QueryRow(ctx, query, conn.SteamID.Int64(), conn.IPAddr.String(), conn.PersonaName, conn.CreatedOn, conn.ServerID).
		Scan(&conn.PersonConnectionID); errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r Repository) GetPlayerMostRecentIP(ctx context.Context, steamID steamid.SteamID) net.IP {
	row, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select("c.ip_addr").
		From("person_connections c").
		Where(sq.Eq{"c.steam_id": steamID.Int64()}).
		OrderBy("c.created_on desc").
		Limit(1))
	if errRow != nil {
		if errors.Is(errRow, database.ErrNoResult) {
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

func (r Repository) GetASNRecordsByNum(ctx context.Context, asNum int64) ([]ASN, error) {
	query := r.Builder().
		Select("cidr::text", "as_num", "as_name").
		From("net_asn").
		Where(sq.Eq{"as_num": asNum})

	rows, errQuery := r.QueryBuilder(ctx, query)
	if errQuery != nil {
		return nil, database.DBErr(errQuery)
	}

	defer rows.Close()

	var records []ASN

	for rows.Next() {
		var asnRecord ASN
		if errScan := rows.
			Scan(&asnRecord.CIDR, &asnRecord.ASNum, &asnRecord.ASName); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		records = append(records, asnRecord)
	}

	return records, nil
}

func (r Repository) GetASNRecordByIP(ctx context.Context, ipAddr netip.Addr) (ASN, error) {
	const query = `
		SELECT ip_range::text, as_num, as_name
		FROM net_asn
		WHERE ip_range >>= $1
		LIMIT 1`

	var asnRecord ASN

	if errQuery := r.QueryRow(ctx, query, ipAddr.String()).
		Scan(&asnRecord.CIDR, &asnRecord.ASNum, &asnRecord.ASName); errQuery != nil {
		return asnRecord, database.DBErr(errQuery)
	}

	return asnRecord, nil
}

func (r Repository) GetLocationRecord(ctx context.Context, ipAddr netip.Addr) (Location, error) {
	const query = `
		SELECT ip_range::text, country_code, country_name, region_name, city_name, ST_Y(location), ST_X(location)
		FROM net_location
		WHERE ip_range >>= $1`

	var record Location

	if errQuery := r.QueryRow(ctx, query, ipAddr.String()).
		Scan(&record.CIDR, &record.CountryCode, &record.CountryName, &record.RegionName,
			&record.CityName, &record.LatLong.Latitude, &record.LatLong.Longitude); errQuery != nil {
		return record, database.DBErr(errQuery)
	}

	return record, nil
}

func (r Repository) GetProxyRecord(ctx context.Context, ipAddr netip.Addr) (Proxy, error) {
	const query = `
		SELECT ip_range::text, proxy_type, country_code, country_name, region_name,
       		city_name, isp, domain_used, usage_type, as_num, as_name, last_seen, threat
		FROM net_proxy
		WHERE ip_range >>= $1`

	var proxyRecord Proxy

	if errQuery := r.QueryRow(ctx, query, ipAddr.String()).
		Scan(&proxyRecord.CIDR, &proxyRecord.ProxyType, &proxyRecord.CountryCode, &proxyRecord.CountryName, &proxyRecord.RegionName, &proxyRecord.CityName, &proxyRecord.ISP,
			&proxyRecord.Domain, &proxyRecord.UsageType, &proxyRecord.ASN, &proxyRecord.AS, &proxyRecord.LastSeen, &proxyRecord.Threat); errQuery != nil {
		return proxyRecord, database.DBErr(errQuery)
	}

	return proxyRecord, nil
}

func (r Repository) LoadASN(ctx context.Context, truncate bool, records []ip2location.ASNRecord) error {
	if truncate {
		slog.Debug("Truncating asn table")

		if errTruncate := r.TruncateTable(ctx, "net_asn"); errTruncate != nil {
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

	batchResults := r.SendBatch(c, &batch)
	if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
		return errors.Join(errCloseBatch, database.ErrCloseBatch)
	}

	return nil
}

func (r Repository) LoadLocation(ctx context.Context, truncate bool, records []ip2location.LocationRecord) error {
	if truncate {
		slog.Debug("Truncating location table")

		if errTruncate := r.TruncateTable(ctx, "net_location"); errTruncate != nil {
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

	batchResults := r.SendBatch(c, &batch)
	if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
		return errors.Join(errCloseBatch, database.ErrCloseBatch)
	}

	return nil
}

func (r Repository) LoadProxies(ctx context.Context, truncate bool, records []ip2location.ProxyRecord) error {
	if truncate {
		slog.Debug("Truncating proxy table")

		if errTruncate := r.TruncateTable(ctx, "net_proxy"); errTruncate != nil {
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

	batchResults := r.SendBatch(c, &batch)
	if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
		return errors.Join(errCloseBatch, database.ErrCloseBatch)
	}

	return nil
}
