package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

// GetBanNet returns the BanNet matching intersecting the supplied ip.
//
// Note that this function does not currently limit results returned. This may change in the future, do not
// rely on this functionality.
func (db *pgStore) GetBanNet(ctx context.Context, ip net.IP) ([]model.BanNet, error) {
	const q = `
		SELECT net_id, cidr, source, created_on, updated_on, reason, valid_until 
		FROM ban_net
		WHERE $1 <<= cidr`
	var nets []model.BanNet
	rows, err := db.c.Query(ctx, q, ip.String())
	if err != nil {
		return nil, Err(err)
	}
	defer rows.Close()
	for rows.Next() {
		var n model.BanNet
		if err2 := rows.Scan(&n.NetID, &n.CIDR, &n.Source, &n.CreatedOn, &n.UpdatedOn, &n.Reason, &n.ValidUntil); err2 != nil {
			return nil, err2
		}
		nets = append(nets, n)
	}
	return nets, nil
}

func (db *pgStore) updateBanNet(ctx context.Context, banNet *model.BanNet) error {
	q, a, e := sb.Update("ban_net").
		Set("cidr", banNet.CIDR).
		Set("source", banNet.Source).
		Set("created_on", banNet.CreatedOn).
		Set("updated_on", banNet.UpdatedOn).
		Set("reason", banNet.Reason).
		Set("valid_until_id", banNet.ValidUntil).
		Where(sq.Eq{"net_id": banNet.NetID}).
		ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return err
	}
	return nil
}

func (db *pgStore) insertBanNet(ctx context.Context, banNet *model.BanNet) error {
	q, a, e := sb.Insert("ban_net").
		Columns("cidr", "source", "created_on", "updated_on", "reason", "valid_until").
		Values(banNet.CIDR, banNet.Source, banNet.CreatedOn, banNet.UpdatedOn, banNet.Reason, banNet.ValidUntil).
		Suffix("RETURNING net_id").
		ToSql()
	if e != nil {
		return e
	}
	err := db.c.QueryRow(ctx, q, a...).Scan(&banNet.NetID)
	if err != nil {
		return err
	}
	return nil
}

func (db *pgStore) SaveBanNet(ctx context.Context, banNet *model.BanNet) error {
	if banNet.NetID > 0 {
		return db.updateBanNet(ctx, banNet)
	}
	return db.insertBanNet(ctx, banNet)
}

func (db *pgStore) DropBanNet(ctx context.Context, ban *model.BanNet) error {
	q, a, e := sb.Delete("ban_net").Where(sq.Eq{"net_id": ban.NetID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return Err(err)
	}
	ban.NetID = 0
	return nil
}

func (db *pgStore) GetExpiredNetBans(ctx context.Context) ([]model.BanNet, error) {
	const q = `
		SELECT net_id, cidr, source, created_on, updated_on, reason, valid_until
		FROM ban_net
		WHERE valid_until < $1`
	var bans []model.BanNet
	rows, err := db.c.Query(ctx, q, config.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.BanNet
		if err2 := rows.Scan(&b.NetID, &b.CIDR, &b.Source, &b.CreatedOn, &b.UpdatedOn, &b.Reason, &b.ValidUntil); err2 != nil {
			return nil, err2
		}
		bans = append(bans, b)
	}
	return bans, nil
}

func (db *pgStore) GetExpiredASNBans(ctx context.Context) ([]model.BanASN, error) {
	const q = `
		SELECT ban_asn_id, as_num, origin, author_id, target_id, reason, valid_until, created_on, updated_on
		FROM ban_asn
		WHERE valid_until < $1`
	var bans []model.BanASN
	rows, err := db.c.Query(ctx, q, config.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.BanASN
		if err2 := rows.Scan(&b.BanASNId, &b.ASNum, &b.Origin, &b.AuthorID, &b.TargetID,
			&b.Reason, &b.ValidUntil, &b.CreatedOn, &b.UpdatedOn); err2 != nil {
			return nil, err2
		}
		bans = append(bans, b)
	}
	return bans, nil
}

func (db *pgStore) GetASNRecordsByNum(ctx context.Context, asNum int64) (ip2location.ASNRecords, error) {
	const q = `
		SELECT ip_from, ip_to, cidr, as_num, as_name 
		FROM net_asn
		WHERE as_num = $1`
	rows, err := db.c.Query(ctx, q, asNum)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records ip2location.ASNRecords
	for rows.Next() {
		var r ip2location.ASNRecord
		if errRow := rows.Scan(&r.IPFrom, &r.IPTo, &r.CIDR, &r.ASNum, &r.ASName); errRow != nil {
			return nil, Err(errRow)
		}
		records = append(records, r)
	}
	return records, nil
}

func (db *pgStore) GetASNRecordByIP(ctx context.Context, ip net.IP, r *ip2location.ASNRecord) error {
	const q = `
		SELECT ip_from, ip_to, cidr, as_num, as_name 
		FROM net_asn
		WHERE $1::inet <@ ip_range
		LIMIT 1`
	if err := db.c.QueryRow(ctx, q, ip.String()).
		Scan(&r.IPFrom, &r.IPTo, &r.CIDR, &r.ASNum, &r.ASName); err != nil {
		return Err(err)
	}
	return nil
}

func (db *pgStore) GetLocationRecord(ctx context.Context, ip net.IP, r *ip2location.LocationRecord) error {
	const q = `
		SELECT ip_from, ip_to, country_code, country_name, region_name, city_name, ST_Y(location), ST_X(location) 
		FROM net_location 
		WHERE $1::inet <@ ip_range`
	if err := db.c.QueryRow(ctx, q, ip.String()).
		Scan(&r.IPFrom, &r.IPTo, &r.CountryCode, &r.CountryName, &r.RegionName, &r.CityName, &r.LatLong.Latitude, &r.LatLong.Longitude); err != nil {
		return Err(err)
	}
	return nil
}

func (db *pgStore) GetProxyRecord(ctx context.Context, ip net.IP, r *ip2location.ProxyRecord) error {
	const q = `
		SELECT ip_from, ip_to, proxy_type, country_code, country_name, region_name, 
       		city_name, isp, domain_used, usage_type, as_num, as_name, last_seen, threat 
		FROM net_proxy 
		WHERE $1::inet <@ ip_range`
	if err := db.c.QueryRow(ctx, q, ip.String()).
		Scan(&r.IPFrom, &r.IPTo, &r.ProxyType, &r.CountryCode, &r.CountryName, &r.RegionName, &r.CityName, &r.ISP,
			&r.Domain, &r.UsageType, &r.ASN, &r.AS, &r.LastSeen, &r.Threat); err != nil {
		return Err(err)
	}
	return nil
}

func (db *pgStore) loadASN(ctx context.Context, records []ip2location.ASNRecord) error {
	t0 := time.Now()
	if err := db.truncateTable(ctx, tableNetASN); err != nil {
		return err
	}
	const q = `
		INSERT INTO net_asn (ip_from, ip_to, cidr, as_num, as_name, ip_range) 
		VALUES($1, $2, $3, $4, $5, iprange($1, $2))`
	b := pgx.Batch{}
	for i, a := range records {
		b.Queue(q, a.IPFrom, a.IPTo, a.CIDR, a.ASNum, a.ASName)
		if i > 0 && i%100000 == 0 || len(records) == i+1 {
			if b.Len() > 0 {
				c, cancel := context.WithTimeout(ctx, time.Second*10)
				r := db.c.SendBatch(c, &b)
				if err := r.Close(); err != nil {
					cancel()
					return err
				}
				cancel()
				b = pgx.Batch{}
				log.Debugf("ASN Progress: %d/%d (%.0f%%)", i, len(records)-1, float64(i)/float64(len(records)-1)*100)
			}
		}
	}
	log.Debugf("Loaded %d ASN4 records in %s", len(records), time.Since(t0).String())
	return nil
}

func (db *pgStore) loadLocation(ctx context.Context, records []ip2location.LocationRecord, _ bool) error {
	t0 := time.Now()
	if err := db.truncateTable(ctx, tableNetLocation); err != nil {
		return err
	}
	const q = `
		INSERT INTO net_location (ip_from, ip_to, country_code, country_name, region_name, city_name, location, ip_range)
		VALUES($1, $2, $3, $4, $5, $6, ST_SetSRID(ST_MakePoint($8, $7), 4326), iprange($1, $2))`
	b := pgx.Batch{}
	for i, a := range records {
		b.Queue(q, a.IPFrom, a.IPTo, a.CountryCode, a.CountryName, a.RegionName, a.CityName, a.LatLong.Latitude, a.LatLong.Longitude)
		if i > 0 && i%100000 == 0 || len(records) == i+1 {
			if b.Len() > 0 {
				c, cancel := context.WithTimeout(ctx, time.Second*10)
				r := db.c.SendBatch(c, &b)
				if err := r.Close(); err != nil {
					cancel()
					return err
				}
				cancel()
				b = pgx.Batch{}
				log.Debugf("Location4 Progress: %d/%d (%.0f%%)", i, len(records)-1, float64(i)/float64(len(records)-1)*100)
			}
		}
	}
	log.Debugf("Loaded %d Location4 records in %s", len(records), time.Since(t0).String())
	return nil
}

func (db *pgStore) loadProxies(ctx context.Context, records []ip2location.ProxyRecord, _ bool) error {
	t0 := time.Now()
	if err := db.truncateTable(ctx, tableNetProxy); err != nil {
		return err
	}
	const q = `
		INSERT INTO net_proxy (ip_from, ip_to, proxy_type, country_code, country_name, region_name, city_name, isp,
		                       domain_used, usage_type, as_num, as_name, last_seen, threat, ip_range)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, iprange($1, $2))`
	b := pgx.Batch{}
	for i, a := range records {
		b.Queue(q, a.IPFrom, a.IPTo, a.ProxyType, a.CountryCode, a.CountryName, a.RegionName, a.CityName,
			a.ISP, a.Domain, a.UsageType, a.ASN, a.AS, a.LastSeen, a.Threat)
		if i > 0 && i%100000 == 0 || len(records) == i+1 {
			if b.Len() > 0 {
				c, cancel := context.WithTimeout(ctx, time.Second*10)
				r := db.c.SendBatch(c, &b)
				if err := r.Close(); err != nil {
					cancel()
					return err
				}
				cancel()
				b = pgx.Batch{}
				log.Debugf("Proxy Progress: %d/%d (%.0f%%)", i, len(records)-1, float64(i)/float64(len(records)-1)*100)
			}
		}
	}
	log.Debugf("Loaded %d Proxy records in %s", len(records), time.Since(t0).String())
	return nil
}

// InsertBlockListData will load the provided datasets into the database
//
// Note that this can take a while on slower machines. For reference it takes
// about ~90s with a local database on a Ryzen 3900X/PCIe4 NVMe SSD.
func (db *pgStore) InsertBlockListData(ctx context.Context, d *ip2location.BlockListData) error {
	if len(d.Proxies) > 0 {
		if err := db.loadProxies(ctx, d.Proxies, false); err != nil {
			return err
		}
	}
	if len(d.Locations4) > 0 {
		if err := db.loadLocation(ctx, d.Locations4, false); err != nil {
			return err
		}
	}
	if len(d.ASN4) > 0 {
		if err := db.loadASN(ctx, d.ASN4); err != nil {
			return err
		}
	}
	return nil
}

func (db *pgStore) GetBanASN(ctx context.Context, asNum int64, banASN *model.BanASN) error {
	const q = `
		SELECT ban_asn_id, as_num, origin, author_id, target_id, reason, valid_until, created_on, updated_on 
		FROM ban_asn 
		WHERE as_num = $1`
	if err := db.c.QueryRow(ctx, q, asNum).Scan(&banASN.BanASNId, &banASN.ASNum, &banASN.Origin, &banASN.AuthorID,
		&banASN.TargetID, &banASN.Reason, &banASN.ValidUntil, &banASN.CreatedOn, &banASN.UpdatedOn); err != nil {
		return Err(err)
	}
	return nil
}

func (db *pgStore) SaveBanASN(ctx context.Context, b *model.BanASN) error {
	b.UpdatedOn = config.Now()
	if b.BanASNId > 0 {
		const q = `
			UPDATE ban_asn 
			SET as_num = $2, origin = $3, author_id = $4, target_id = $5, reason = $6,
				valid_until = $7, updated_on = $8
			WHERE ban_asn_id = $1`

		_, errUpd := db.c.Exec(ctx, q, b.BanASNId, b.ASNum, b.Origin, b.AuthorID, b.TargetID,
			b.Reason, b.ValidUntil, b.UpdatedOn)
		return Err(errUpd)

	}
	const qi = `
		INSERT INTO ban_asn (as_num, origin, author_id, target_id, reason, valid_until, updated_on, created_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING ban_asn_id`
	errIns := db.c.QueryRow(ctx, qi, b.ASNum, b.Origin, b.AuthorID, b.TargetID,
		b.Reason, b.ValidUntil, b.UpdatedOn, b.CreatedOn).Scan(&b.BanASNId)
	return Err(errIns)
}

func (db *pgStore) DropBanASN(ctx context.Context, ban *model.BanASN) error {
	const q = `DELETE FROM ban_asn WHERE ban_asn_id = $1`
	_, err := db.c.Exec(ctx, q, ban.BanASNId)
	return Err(err)
}
