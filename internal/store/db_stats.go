package store

import (
	"context"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sync"
)

func (database *pgStore) GetStats(ctx context.Context, stats *model.Stats) error {
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
	if errQuery := database.conn.QueryRow(ctx, q).
		Scan(&stats.BansTotal, &stats.BansDay, &stats.BansWeek, &stats.BansMonth,
			&stats.Bans3Month, &stats.Bans6Month, &stats.BansYear, &stats.BansCIDRTotal,
			&stats.AppealsOpen, &stats.AppealsClosed, &stats.FilteredWords, &stats.ServersTotal,
		); errQuery != nil {
		log.Errorf("Failed to fetch stats: %v", errQuery)
		return Err(errQuery)
	}
	return nil

}

type statResult struct {
	result *int64
	query  statQueryOpts
}

// GetPlayerStats calculates and returns basic stats for a player using the server_log events
// FIXME Since we currently run on high-core count hardware with nvme drives
// we are running the queries concurrently for now
func (database *pgStore) GetPlayerStats(ctx context.Context, sid steamid.SID64, stats *model.PlayerStats) error {
	waitGroup := &sync.WaitGroup{}
	rwMutex := &sync.RWMutex{}
	queries := []statResult{
		{&stats.Damage, statQueryOpts{sourceId: sid, msgTypes: []logparse.EventType{logparse.Damage}, sumColumn: "damage"}},
		{&stats.DamageTaken, statQueryOpts{targetId: sid, msgTypes: []logparse.EventType{logparse.Damage}, sumColumn: "damage"}},
		{&stats.Healing, statQueryOpts{sourceId: sid, msgTypes: []logparse.EventType{logparse.Healed}, sumColumn: "healing"}},
		{&stats.Shots, statQueryOpts{sourceId: sid, msgTypes: []logparse.EventType{logparse.ShotFired}, countColumn: "*"}},
		{&stats.Hits, statQueryOpts{sourceId: sid, msgTypes: []logparse.EventType{logparse.ShotHit}, countColumn: "*"}},
		{&stats.Kills, statQueryOpts{sourceId: sid, msgTypes: []logparse.EventType{logparse.Killed}, countColumn: "*"}},
		{&stats.Deaths, statQueryOpts{targetId: sid, msgTypes: []logparse.EventType{logparse.Killed}, countColumn: "*"}},
		{&stats.Assists, statQueryOpts{sourceId: sid, msgTypes: []logparse.EventType{logparse.KillAssist}, countColumn: "*"}},
	}
	waitGroup.Add(len(queries))
	for _, query := range queries {
		go func(v *int64, q statQueryOpts) {
			defer waitGroup.Done()
			dmg, errDmg := database.getEventSum(ctx, q)
			if errDmg != nil {
				log.Warnf("Failed to get player damage")
			}
			rwMutex.Lock()
			*v = dmg
			rwMutex.Unlock()
		}(query.result, query.query)
	}
	waitGroup.Wait()
	return nil
}

type statQueryOpts struct {
	sourceId steamid.SID64
	targetId steamid.SID64
	serverId int64
	msgTypes []logparse.EventType
	//since       *time.Time
	sumColumn   string
	countColumn string
}

func (database *pgStore) getEventSum(ctx context.Context, opts statQueryOpts) (int64, error) {
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
	query, args, errQueryArgs := qb.Where(ands).ToSql()
	if errQueryArgs != nil {
		return 0, errors.Wrapf(errQueryArgs, "Failed to to generate query")
	}
	log.Tracef("getEventSum: %s", query)
	//const q = `SELECT sum(s.damage) as total FROM server_log s WHERE s.source_id = $1 AND event_type = $2`
	var value int64
	if errQueryRow := database.conn.QueryRow(ctx, query, args...).Scan(&value); errQueryRow != nil {
		return 0, errors.Wrapf(errQueryRow, "Failed to fetch player result sum")
	}
	return value, nil
}

func (database *pgStore) GetGlobalStats(ctx context.Context, stats *model.GlobalStats) error {
	waitGroup := &sync.WaitGroup{}
	rwMutex := &sync.RWMutex{}
	queries := []statResult{
		{&stats.Damage, statQueryOpts{msgTypes: []logparse.EventType{logparse.Damage}, sumColumn: "damage"}},
		{&stats.Healing, statQueryOpts{msgTypes: []logparse.EventType{logparse.Damage}, sumColumn: "healing"}},
		{&stats.Shots, statQueryOpts{msgTypes: []logparse.EventType{logparse.ShotFired}, countColumn: "*"}},
		{&stats.Hits, statQueryOpts{msgTypes: []logparse.EventType{logparse.ShotHit}, countColumn: "*"}},
		{&stats.Kills, statQueryOpts{msgTypes: []logparse.EventType{logparse.Killed}, countColumn: "*"}},
		{&stats.Assists, statQueryOpts{msgTypes: []logparse.EventType{logparse.KillAssist}, countColumn: "*"}},
	}
	waitGroup.Add(len(queries))
	for _, query := range queries {
		go func(v *int64, q statQueryOpts) {
			defer waitGroup.Done()
			value, errStat := database.getEventSum(ctx, q)
			if errStat != nil {
				log.Warnf("Failed to get stat value: %v", errStat)
			}
			rwMutex.Lock()
			*v = value
			rwMutex.Unlock()
		}(query.result, query.query)
	}
	waitGroup.Wait()
	return nil
}

func (database *pgStore) GetServerStats(ctx context.Context, serverId int64, stats *model.ServerStats) error {
	waitGroup := &sync.WaitGroup{}
	rwMutex := &sync.RWMutex{}
	queries := []statResult{
		{&stats.Damage, statQueryOpts{serverId: serverId, msgTypes: []logparse.EventType{logparse.Damage}, sumColumn: "damage"}},
		{&stats.Healing, statQueryOpts{serverId: serverId, msgTypes: []logparse.EventType{logparse.Damage}, sumColumn: "healing"}},
		{&stats.Shots, statQueryOpts{serverId: serverId, msgTypes: []logparse.EventType{logparse.ShotFired}, countColumn: "*"}},
		{&stats.Hits, statQueryOpts{serverId: serverId, msgTypes: []logparse.EventType{logparse.ShotHit}, countColumn: "*"}},
		{&stats.Kills, statQueryOpts{serverId: serverId, msgTypes: []logparse.EventType{logparse.Killed}, countColumn: "*"}},
		{&stats.Assists, statQueryOpts{serverId: serverId, msgTypes: []logparse.EventType{logparse.KillAssist}, countColumn: "*"}},
	}
	waitGroup.Add(len(queries))
	for _, query := range queries {
		go func(v *int64, q statQueryOpts) {
			defer waitGroup.Done()
			value, errStat := database.getEventSum(ctx, q)
			if errStat != nil {
				log.Warnf("Failed to get stat value: %v", errStat)
			}
			rwMutex.Lock()
			*v = value
			rwMutex.Unlock()
		}(query.result, query.query)
	}
	waitGroup.Wait()
	return nil
}
func (database *pgStore) GetReplayLogs(ctx context.Context, offset uint64, limit uint64) ([]model.ServerEvent, error) {
	const query = `
			SELECT 
			    l.log_id, l.event_type, l.created_on,
				srv.server_id, srv.short_name,
				l.source_id, src.personaname, src.avatarfull, src.avatar,
			    l.target_id, tar.personaname, tar.avatarfull, tar.avatar,
				l.weapon, l.damage, l.attacker_position, l.victim_position, l.assister_position,
				l.item, l.player_class, l.player_team, l.meta_data, l.healing 
			FROM server_log l
			LEFT JOIN server srv on srv.server_id = l.server_id
			LEFT JOIN person src on src.steam_id = l.source_id
			LEFT JOIN person tar on tar.steam_id = l.target_id
			ORDER BY l.created_on DESC
			OFFSET %d 
			LIMIT %d`
	rows, errQuery := database.Query(ctx, fmt.Sprintf(query, offset, limit))
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	var localResults []model.ServerEvent
	for rows.Next() {
		event := model.ServerEvent{
			Server: &model.Server{},
			Source: &model.Person{PlayerSummary: &steamweb.PlayerSummary{}},
			Target: &model.Person{PlayerSummary: &steamweb.PlayerSummary{}},
		}
		if errScan := rows.Scan(
			&event.LogID, &event.EventType, &event.CreatedOn,
			&event.Server.ServerID, &event.Server.ServerNameShort,
			&event.Source.SteamID, &event.Source.PersonaName, &event.Source.AvatarFull, &event.Source.Avatar,
			&event.Target.SteamID, &event.Target.PersonaName, &event.Target.AvatarFull, &event.Target.Avatar,
			&event.Weapon, &event.Damage, &event.AttackerPOS, &event.VictimPOS, &event.AssisterPOS,
			&event.Item, &event.PlayerClass, &event.Team, &event.MetaData, &event.Healing); errScan != nil {
			return nil, Err(errScan)
		}
		localResults = append(localResults, event)
	}
	return localResults, nil
}
