package store

import (
	"context"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

func (db *pgStore) GetStats(ctx context.Context, stats *model.Stats) error {
	const q = `
	SELECT 
		(SELECT COUNT(ban_id) FROM ban) as bans_total,
		(SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 DAY')) as bans_day,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 DAY')) as bans_week,
		(SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 MONTH')) as bans_month, 
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '3 MONTH')) as bans_3month,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '6 MONTH')) as bans_6month,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 YEAR')) as bans_year,
		(SELECT COUNT(net_id) FROM ban_net) as bans_cidr, 
		(SELECT COUNT(appeal_id) FROM ban_appeal WHERE appeal_state = 0) as appeals_open,
		(SELECT COUNT(appeal_id) FROM ban_appeal WHERE appeal_state = 1 OR appeal_state = 2) as appeals_closed,
		(SELECT COUNT(word_id) FROM filtered_word) as filtered_words,
		(SELECT COUNT(server_id) FROM server) as servers_total`
	if err := db.c.QueryRow(ctx, q).
		Scan(&stats.BansTotal, &stats.BansDay, &stats.BansWeek, &stats.BansMonth,
			&stats.Bans3Month, &stats.Bans6Month, &stats.BansYear, &stats.BansCIDRTotal,
			&stats.AppealsOpen, &stats.AppealsClosed, &stats.FilteredWords, &stats.ServersTotal,
		); err != nil {
		log.Errorf("Failed to fetch stats: %v", err)
		return dbErr(err)
	}
	return nil

}

type GlobalStats struct {
	UniquePlayers int64
	Kills         int64
	Assists       int64
	Damage        int64
	Healing       int64
	Shots         int64
	Hits          int64
}

type PlayerStats struct {
	Kills       int64
	Assists     int64
	Deaths      int64
	Damage      int64
	DamageTaken int64
	Healing     int64
	Shots       int64
	Hits        int64
	Games       int64
	Wins        int64
	Losses      int64
}

type ServerStats struct {
	Kills   int64
	Assists int64
	Damage  int64
	Healing int64
	Shots   int64
	Hits    int64
}

type statResult struct {
	result *int64
	query  statQueryOpts
}

// GetPlayerStats calculates and returns basic stats for a player using the server_log events
// FIXME Since we currently run on high-core count hardware with nvme drives
// we are running the queries concurrently for now
func (db *pgStore) GetPlayerStats(ctx context.Context, sid steamid.SID64) (PlayerStats, error) {
	var stats PlayerStats
	wg := &sync.WaitGroup{}
	mu := &sync.RWMutex{}
	queries := []statResult{
		{&stats.Damage, statQueryOpts{sourceId: sid, msgTypes: []logparse.MsgType{logparse.Damage}, sumColumn: "damage"}},
		{&stats.DamageTaken, statQueryOpts{targetId: sid, msgTypes: []logparse.MsgType{logparse.Damage}, sumColumn: "damage"}},
		{&stats.Healing, statQueryOpts{sourceId: sid, msgTypes: []logparse.MsgType{logparse.Damage}, sumColumn: "healing"}},
		{&stats.Shots, statQueryOpts{sourceId: sid, msgTypes: []logparse.MsgType{logparse.ShotFired}, countColumn: "*"}},
		{&stats.Hits, statQueryOpts{sourceId: sid, msgTypes: []logparse.MsgType{logparse.ShotHit}, countColumn: "*"}},
		{&stats.Kills, statQueryOpts{sourceId: sid, msgTypes: []logparse.MsgType{logparse.Killed}, countColumn: "*"}},
		{&stats.Deaths, statQueryOpts{targetId: sid, msgTypes: []logparse.MsgType{logparse.Killed}, countColumn: "*"}},
		{&stats.Assists, statQueryOpts{sourceId: sid, msgTypes: []logparse.MsgType{logparse.KillAssist}, countColumn: "*"}},
	}
	wg.Add(len(queries))
	for _, query := range queries {
		go func(v *int64, q statQueryOpts) {
			defer wg.Done()
			dmg, errDmg := db.getEventSum(ctx, q)
			if errDmg != nil {
				log.Warnf("Failed to get player damage")
			}
			mu.Lock()
			*v = dmg
			mu.Unlock()
		}(query.result, query.query)
	}
	wg.Wait()
	return stats, nil
}

type statQueryOpts struct {
	sourceId    steamid.SID64
	targetId    steamid.SID64
	serverId    int64
	msgTypes    []logparse.MsgType
	since       *time.Time
	sumColumn   string
	countColumn string
}

func (db *pgStore) getEventSum(ctx context.Context, opts statQueryOpts) (int64, error) {
	var qb squirrel.SelectBuilder
	if opts.sumColumn != "" && opts.countColumn != "" {
		return 0, errors.New("sumColumn and countColumn are mutually exclusive")
	} else if opts.sumColumn != "" {
		qb = sb.Select(fmt.Sprintf("SUM(s.%s) as result", opts.sumColumn))
	} else {
		qb = sb.Select("COUNT(s.*) as result")
	}
	var ands squirrel.And
	if opts.serverId > 0 {
		ands = append(ands, squirrel.Eq{"s.server_id": opts.serverId})
	}
	if opts.sourceId != 0 {
		ands = append(ands, squirrel.Eq{"s.source_id": opts.sourceId})
	}
	if opts.targetId != 0 {
		ands = append(ands, squirrel.Eq{"s.target_id": opts.targetId})
	}
	var mTypes squirrel.Or
	for _, mt := range opts.msgTypes {
		mTypes = append(mTypes, squirrel.Eq{"event_type": mt})
	}
	qb = qb.From("server_log s")
	ands = append(ands, mTypes)
	query, args, errQuery := qb.Where(ands).ToSql()
	if errQuery != nil {
		return 0, errors.Wrapf(errQuery, "Failed to to generate query")
	}
	log.Tracef("getEventSum: %s", query)
	//const q = `SELECT sum(s.damage) as total FROM server_log s WHERE s.source_id = $1 AND event_type = $2`
	var value int64
	if err := db.c.QueryRow(ctx, query, args...).Scan(&value); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player result sum")
	}
	return value, nil
}

func (db *pgStore) GetGlobalStats(ctx context.Context) (GlobalStats, error) {
	var stats GlobalStats
	wg := &sync.WaitGroup{}
	mu := &sync.RWMutex{}
	queries := []statResult{
		{&stats.Damage, statQueryOpts{msgTypes: []logparse.MsgType{logparse.Damage}, sumColumn: "damage"}},
		{&stats.Healing, statQueryOpts{msgTypes: []logparse.MsgType{logparse.Damage}, sumColumn: "healing"}},
		{&stats.Shots, statQueryOpts{msgTypes: []logparse.MsgType{logparse.ShotFired}, countColumn: "*"}},
		{&stats.Hits, statQueryOpts{msgTypes: []logparse.MsgType{logparse.ShotHit}, countColumn: "*"}},
		{&stats.Kills, statQueryOpts{msgTypes: []logparse.MsgType{logparse.Killed}, countColumn: "*"}},
		{&stats.Assists, statQueryOpts{msgTypes: []logparse.MsgType{logparse.KillAssist}, countColumn: "*"}},
	}
	wg.Add(len(queries))
	for _, query := range queries {
		go func(v *int64, q statQueryOpts) {
			defer wg.Done()
			value, errStat := db.getEventSum(ctx, q)
			if errStat != nil {
				log.Warnf("Failed to get stat value: %v", errStat)
			}
			mu.Lock()
			*v = value
			mu.Unlock()
		}(query.result, query.query)
	}
	wg.Wait()
	return stats, nil
}

func (db *pgStore) GetServerStats(ctx context.Context, serverId int64) (ServerStats, error) {
	var stats ServerStats
	wg := &sync.WaitGroup{}
	mu := &sync.RWMutex{}
	queries := []statResult{
		{&stats.Damage, statQueryOpts{serverId: serverId, msgTypes: []logparse.MsgType{logparse.Damage}, sumColumn: "damage"}},
		{&stats.Healing, statQueryOpts{serverId: serverId, msgTypes: []logparse.MsgType{logparse.Damage}, sumColumn: "healing"}},
		{&stats.Shots, statQueryOpts{serverId: serverId, msgTypes: []logparse.MsgType{logparse.ShotFired}, countColumn: "*"}},
		{&stats.Hits, statQueryOpts{serverId: serverId, msgTypes: []logparse.MsgType{logparse.ShotHit}, countColumn: "*"}},
		{&stats.Kills, statQueryOpts{serverId: serverId, msgTypes: []logparse.MsgType{logparse.Killed}, countColumn: "*"}},
		{&stats.Assists, statQueryOpts{serverId: serverId, msgTypes: []logparse.MsgType{logparse.KillAssist}, countColumn: "*"}},
	}
	wg.Add(len(queries))
	for _, query := range queries {
		go func(v *int64, q statQueryOpts) {
			defer wg.Done()
			value, errStat := db.getEventSum(ctx, q)
			if errStat != nil {
				log.Warnf("Failed to get stat value: %v", errStat)
			}
			mu.Lock()
			*v = value
			mu.Unlock()
		}(query.result, query.query)
	}
	wg.Wait()
	return stats, nil
}
