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

type statResult struct {
	result *int64
	query  statQueryOpts
}

// GetPlayerStats calculates and returns basic stats for a player using the server_log events
// FIXME Since we currently run on high-core count hardware with nvme drives
// we are running the queries concurrently for now
func (db *pgStore) GetPlayerStats(ctx context.Context, sid steamid.SID64, stats *model.PlayerStats) error {
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
	return nil
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

func (db *pgStore) GetGlobalStats(ctx context.Context, stats *model.GlobalStats) error {
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
	return nil
}

func (db *pgStore) GetServerStats(ctx context.Context, serverId int64, stats *model.ServerStats) error {
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
	return nil
}

func (db *pgStore) RebuildStats(ctx context.Context) error {
	const limit = 100_000
	var fetchLogs = func(offset uint64) ([]model.ServerEvent, error) {
		const q = `
			SELECT 
			    l.log_id, l.event_type, l.created_on,
				srv.server_id, srv.short_name,
				l.source_id, src.personaname, src.avatarfull, src.avatar,
			    l.target_id, tar.personaname, tar.avatarfull, tar.avatar,
				l.weapon, l.damage, l.attacker_position, l.victim_position, l.assister_position,
				l.item, l.extra, l.player_class, l.player_team, l.meta_data, l.healing 
			FROM server_log l
			LEFT JOIN server srv on srv.server_id = l.server_id
			LEFT JOIN person src on src.steam_id = l.source_id
			LEFT JOIN person tar on tar.steam_id = l.target_id
			ORDER BY l.created_on DESC
			OFFSET ? 
			LIMIT ?`
		rows, errQuery := db.c.Query(ctx, q, offset, limit)
		if errQuery != nil {
			return nil, dbErr(errQuery)
		}
		defer rows.Close()
		var localResults []model.ServerEvent
		for rows.Next() {
			e := model.ServerEvent{
				Server: &model.Server{},
				Source: &model.Person{PlayerSummary: &steamweb.PlayerSummary{}},
				Target: &model.Person{PlayerSummary: &steamweb.PlayerSummary{}},
			}
			if errScan := rows.Scan(
				&e.LogID, &e.EventType, &e.CreatedOn,
				&e.Server.ServerID, &e.Server.ServerName,
				&e.Source.SteamID, &e.Source.PersonaName, &e.Source.AvatarFull, &e.Source.Avatar,
				&e.Target.SteamID, &e.Target.PersonaName, &e.Target.AvatarFull, &e.Target.Avatar,
				&e.Weapon, &e.Damage, &e.AttackerPOS, &e.VictimPOS, &e.AssisterPOS,
				&e.Item, &e.Extra, &e.PlayerClass, &e.Team, &e.MetaData, &e.Healing); errScan != nil {
				return nil, dbErr(errScan)
			}
			localResults = append(localResults, e)
		}
		return localResults, nil
	}
	var (
		idx     uint64 = 0
		offset  uint64 = 0
		results []model.ServerEvent
	)
	for {
		logs, fetchErr := fetchLogs(offset * idx)
		if fetchErr != nil {
			break
		}
		results = append(results, logs...)
		idx++
	}

	players := map[steamid.SID64]*model.PlayerStats{}
	servers := map[int64]*model.ServerStats{}
	global := &model.GlobalStats{}

	var getPlayer = func(sid steamid.SID64) (*model.PlayerStats, bool) {
		v, f := players[sid]
		if !f {
			v = &model.PlayerStats{}
			players[sid] = v
		}
		return v, f
	}

	var getServer = func(sid int64) *model.ServerStats {
		v, f := servers[sid]
		if !f {
			v = &model.ServerStats{}
			servers[sid] = v
		}
		return v
	}

	// TODO
	// - individual game state cache to track who is on winning team
	// - Filter out pre-game events likes kills/damage
	// -

	for _, event := range results {
		source, sourceExists := getPlayer(event.Source.SteamID)
		target, _ := getPlayer(event.Target.SteamID)
		server := getServer(event.Server.ServerID)
		if !sourceExists {
			global.UniquePlayers++
		}
		switch event.EventType {
		case logparse.JoinedTeam:
		// Track game team for wins
		case logparse.KillAssist:
			global.Assists++
			server.Assists++
			source.Assists++
		case logparse.Healed:
			global.Healing++
			server.Healing++
			source.Healing++
		case logparse.Connected:
		// Add player to game state, remove
		case logparse.Disconnected:
		// remove player from game state
		case logparse.Say:
			global.Messages++
			server.Messages++
			source.Messages++
		case logparse.SayTeam:
			global.MessagesTeam++
			server.MessagesTeam++
			source.MessagesTeam++
		case logparse.MedicDeath:
			// Count drops
			uberPct, ok := event.MetaData["uber"].(int64)
			if ok && uberPct >= 100 {
				global.MedicDroppedUber++
				server.MedicDroppedUber++
				source.MedicDroppedUber++
			}
		case logparse.WTeamFinalScore:
		// Win/loss rates
		case logparse.BuiltObject:
			// bob the builders
			global.ObjectBuilt++
			server.ObjectBuilt++
			source.ObjectBuilt++
		case logparse.CaptureBlocked:
			// Blocks
			global.PointDefends++
			server.PointDefends++
			source.PointDefends++
		case logparse.PointCaptured:
			// captures, multiple people
			global.PointCaptures++
			server.PointCaptures++
			// TODO calc all people
			source.PointCaptures++
		case logparse.Domination:
			global.Dominations++
			server.Dominations++
			source.Dominations++
			target.Dominated++
		case logparse.Revenge:
			global.Revenges++
			server.Revenges++
			source.Suicides++
		case logparse.Suicide:
			global.Suicides++
			server.Suicides++
			source.Suicides++
		case logparse.WRoundWin:
		case logparse.WRoundLen:
		case logparse.Extinguished:
			global.Extinguishes++
			server.Extinguishes++
			source.Extinguishes++
		case logparse.SpawnedAs:
			switch event.PlayerClass {
			case logparse.Scout:
				global.SpawnScout++
				server.SpawnScout++
				source.SpawnScout++
			case logparse.Soldier:
				global.SpawnSoldier++
				server.SpawnSoldier++
				source.SpawnSoldier++
			case logparse.Pyro:
				global.SpawnPyro++
				server.SpawnPyro++
				source.SpawnPyro++
			case logparse.Demo:
				global.SpawnDemo++
				server.SpawnDemo++
				source.SpawnDemo++
			case logparse.Heavy:
				global.SpawnHeavy++
				server.SpawnHeavy++
				source.SpawnHeavy++
			case logparse.Engineer:
				global.SpawnEngineer++
				server.SpawnEngineer++
				source.SpawnEngineer++
			case logparse.Medic:
				global.SpawnMedic++
				server.SpawnMedic++
				source.SpawnMedic++
			case logparse.Sniper:
				global.SpawnSniper++
				server.SpawnSniper++
				source.SpawnSniper++
			case logparse.Spy:
				global.SpawnSpy++
				server.SpawnSpy++
				source.SpawnSpy++
			}
		case logparse.Pickup:
			switch event.Item {
			case logparse.ItemAmmoLarge:
				source.PickupAmmoLarge++
				server.PickupAmmoLarge++
				global.PickupAmmoLarge++
			case logparse.ItemAmmoMedium:
				source.PickupAmmoMedium++
				server.PickupAmmoMedium++
				global.PickupAmmoMedium++
			case logparse.ItemAmmoSmall:
				source.PickupAmmoSmall++
				server.PickupAmmoSmall++
				global.PickupAmmoSmall++
			case logparse.ItemHPLarge:
				source.PickupHPLarge++
				server.PickupHPLarge++
				global.PickupHPLarge++
			case logparse.ItemHPMedium:
				source.PickupHPMedium++
				server.PickupHPMedium++
				global.PickupHPMedium++
			case logparse.ItemHPSmall:
				source.PickupHPSmall++
				server.PickupHPSmall++
				global.PickupHPSmall++
			}
		case logparse.ShotFired:
			global.Shots++
			source.Shots++
			server.Shots++
		case logparse.ShotHit:
			global.Shots++
			source.Shots++
			server.Shots++
		case logparse.Killed:
			global.Kills++
			source.Kills++
			target.Deaths++
			server.Kills++
		case logparse.Damage:
			global.Damage += event.Damage
			source.Damage += event.Damage
			target.DamageTaken += event.Damage
			server.Damage += event.Damage
		}
	}
	return nil
}
