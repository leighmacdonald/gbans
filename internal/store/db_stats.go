package store

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sync"
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
}

type ServerStats struct {
	Kills   int64
	Assists int64
	Damage  int64
	Healing int64
	Shots   int64
	Hits    int64
}

// GetPlayerStats calculates and returns basic stats for a player using the server_log events
// FIXME Since we currently run on high-core count hardware with nvme drives
// we are running the queries concurrently for now
func (db *pgStore) GetPlayerStats(ctx context.Context, sid steamid.SID64) (PlayerStats, error) {
	var stats PlayerStats
	wg := &sync.WaitGroup{}
	mu := &sync.RWMutex{}

	wg.Add(8)
	go func() {
		defer wg.Done()
		dmg, errDmg := db.getPlayerDamage(ctx, sid)
		if errDmg != nil {
			log.Warnf("Failed to get player damage")
		}
		mu.Lock()
		stats.Damage = dmg
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		dmg, errDmg := db.getPlayerDamageTaken(ctx, sid)
		if errDmg != nil {
			log.Warnf("Failed to get player damage taken")
		}
		mu.Lock()
		stats.DamageTaken = dmg
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		healing, errDmg := db.getPlayerHealing(ctx, sid)
		if errDmg != nil {
			log.Warnf("Failed to get player healing")
		}
		mu.Lock()
		stats.Healing = healing
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getPlayerEventCount(ctx, sid, logparse.ShotFired)
		if errDmg != nil {
			log.Warnf("Failed to get player shits fired")
		}
		mu.Lock()
		stats.Shots = count
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getPlayerEventCount(ctx, sid, logparse.ShotHit)
		if errDmg != nil {
			log.Warnf("Failed to get player shots hit")
		}
		mu.Lock()
		stats.Hits = count
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getPlayerEventCount(ctx, sid, logparse.Killed)
		if errDmg != nil {
			log.Warnf("Failed to get player kills")
		}
		mu.Lock()
		stats.Kills = count
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getPlayerEventTargetCount(ctx, sid, logparse.Killed)
		if errDmg != nil {
			log.Warnf("Failed to get player deaths")
		}
		mu.Lock()
		stats.Deaths = count
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getPlayerEventCount(ctx, sid, logparse.KillAssist)
		if errDmg != nil {
			log.Warnf("Failed to get player assists")
		}
		mu.Lock()
		stats.Assists = count
		mu.Unlock()
	}()
	wg.Wait()
	return stats, nil
}

func (db *pgStore) getPlayerDamage(ctx context.Context, sid steamid.SID64) (int64, error) {
	const q = `SELECT sum(s.damage) as total FROM server_log s WHERE s.source_id = $1 AND event_type = $2`
	var dmg int64
	if err := db.c.QueryRow(ctx, q, sid.Int64(), logparse.Damage).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player damage sum")
	}
	return dmg, nil
}

func (db *pgStore) getMedicDrops(ctx context.Context, sid steamid.SID64) (int64, error) {
	const q = `SELECT count(*) as total FROM server_log s WHERE s.source_id = $1 AND event_type = $2`
	var dmg int64
	if err := db.c.QueryRow(ctx, q, sid.Int64(), logparse.MedicDeath).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player damage sum")
	}
	return dmg, nil
}

func (db *pgStore) getMedicUses(ctx context.Context, sid steamid.SID64) (int64, error) {
	const q = `SELECT sum(s.damage) as total FROM server_log s WHERE s.source_id = $1 AND event_type = $2`
	var dmg int64
	if err := db.c.QueryRow(ctx, q, sid.Int64(), logparse.Damage).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player damage sum")
	}
	return dmg, nil
}

func (db *pgStore) getPlayerDamageTaken(ctx context.Context, sid steamid.SID64) (int64, error) {
	const q = `SELECT sum(s.damage) as total FROM server_log s WHERE s.target_id = $1`
	var dmg int64
	if err := db.c.QueryRow(ctx, q, sid.Int64()).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player damage taken sum")
	}
	return dmg, nil
}

func (db *pgStore) getPlayerHealing(ctx context.Context, sid steamid.SID64) (int64, error) {
	const q = `SELECT sum(s.healing) as total FROM server_log s WHERE s.source_id = $1`
	var dmg int64
	if err := db.c.QueryRow(ctx, q, sid.Int64()).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player healing sum")
	}
	return dmg, nil
}

func (db *pgStore) getPlayerEventCount(ctx context.Context, sid steamid.SID64, event logparse.MsgType) (int64, error) {
	const q = `SELECT count(*) as total FROM server_log s WHERE s.source_id = $1 AND event_type = $2`
	var count int64
	if err := db.c.QueryRow(ctx, q, sid.Int64(), event).Scan(&count); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player event count")
	}
	return count, nil
}

func (db *pgStore) getPlayerEventTargetCount(ctx context.Context, sid steamid.SID64, event logparse.MsgType) (int64, error) {
	const q = `SELECT count(*) as total FROM server_log s WHERE s.target_id = $1 AND event_type = $2`
	var count int64
	if err := db.c.QueryRow(ctx, q, sid.Int64(), event).Scan(&count); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player event target count")
	}
	return count, nil
}

func (db *pgStore) GetGlobalStats(ctx context.Context) (GlobalStats, error) {
	var stats GlobalStats
	wg := &sync.WaitGroup{}
	mu := &sync.RWMutex{}

	wg.Add(6)
	go func() {
		defer wg.Done()
		dmg, errDmg := db.getGlobalDamage(ctx)
		if errDmg != nil {
			log.Warnf("Failed to get player damage")
		}
		mu.Lock()
		stats.Damage = dmg
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		healing, errDmg := db.getGlobalHealing(ctx)
		if errDmg != nil {
			log.Warnf("Failed to get player healing")
		}
		mu.Lock()
		stats.Healing = healing
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getGlobalEventCount(ctx, logparse.ShotFired)
		if errDmg != nil {
			log.Warnf("Failed to get player shits fired")
		}
		mu.Lock()
		stats.Shots = count
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getGlobalEventCount(ctx, logparse.ShotHit)
		if errDmg != nil {
			log.Warnf("Failed to get player shots hit")
		}
		mu.Lock()
		stats.Hits = count
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getGlobalEventCount(ctx, logparse.Killed)
		if errDmg != nil {
			log.Warnf("Failed to get player kills")
		}
		mu.Lock()
		stats.Kills = count
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getGlobalEventCount(ctx, logparse.KillAssist)
		if errDmg != nil {
			log.Warnf("Failed to get player assists")
		}
		mu.Lock()
		stats.Assists = count
		mu.Unlock()
	}()
	wg.Wait()
	return stats, nil
}

func (db *pgStore) GetServerStats(ctx context.Context, sid int64) (ServerStats, error) {
	var stats ServerStats
	wg := &sync.WaitGroup{}
	mu := &sync.RWMutex{}

	wg.Add(6)
	go func() {
		defer wg.Done()
		dmg, errDmg := db.getServerDamage(ctx, sid)
		if errDmg != nil {
			log.Warnf("Failed to get player damage")
		}
		mu.Lock()
		stats.Damage = dmg
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		healing, errDmg := db.getServerHealing(ctx, sid)
		if errDmg != nil {
			log.Warnf("Failed to get player healing")
		}
		mu.Lock()
		stats.Healing = healing
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getServerEventCount(ctx, sid, logparse.ShotFired)
		if errDmg != nil {
			log.Warnf("Failed to get player shits fired")
		}
		mu.Lock()
		stats.Shots = count
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getServerEventCount(ctx, sid, logparse.ShotHit)
		if errDmg != nil {
			log.Warnf("Failed to get player shots hit")
		}
		mu.Lock()
		stats.Hits = count
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getServerEventCount(ctx, sid, logparse.Killed)
		if errDmg != nil {
			log.Warnf("Failed to get player kills")
		}
		mu.Lock()
		stats.Kills = count
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		count, errDmg := db.getServerEventCount(ctx, sid, logparse.KillAssist)
		if errDmg != nil {
			log.Warnf("Failed to get player assists")
		}
		mu.Lock()
		stats.Assists = count
		mu.Unlock()
	}()
	wg.Wait()
	return stats, nil
}

func (db *pgStore) getServerDamage(ctx context.Context, sid int64) (int64, error) {
	const q = `SELECT sum(s.damage) as total FROM server_log s WHERE s.server_id = $1 AND event_type = $2`
	var dmg int64
	if err := db.c.QueryRow(ctx, q, sid, logparse.Damage).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player damage sum")
	}
	return dmg, nil
}

func (db *pgStore) getServerMedicDrops(ctx context.Context, sid int64) (int64, error) {
	const q = `SELECT count(*) as total FROM server_log s WHERE s.server_id = $1 AND event_type = $2`
	var dmg int64
	if err := db.c.QueryRow(ctx, q, sid, logparse.MedicDeath).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch server medic drops sum")
	}
	return dmg, nil
}

func (db *pgStore) getServerUses(ctx context.Context, sid int64) (int64, error) {
	const q = `SELECT count(*) as total FROM server_log s WHERE s.server_id = $1 AND event_type = $2`
	var dmg int64
	if err := db.c.QueryRow(ctx, q, sid, logparse.EmptyUber).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player damage sum")
	}
	return dmg, nil
}

func (db *pgStore) getServerHealing(ctx context.Context, sid int64) (int64, error) {
	const q = `SELECT sum(s.healing) as total FROM server_log s WHERE s.server_id = $1`
	var dmg int64
	if err := db.c.QueryRow(ctx, q, sid).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player healing sum")
	}
	return dmg, nil
}

func (db *pgStore) getServerEventCount(ctx context.Context, sid int64, event logparse.MsgType) (int64, error) {
	const q = `SELECT count(*) as total FROM server_log s WHERE s.server_id = $1 AND event_type = $2`
	var count int64
	if err := db.c.QueryRow(ctx, q, sid, event).Scan(&count); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player event count")
	}
	return count, nil
}

func (db *pgStore) getServerEventTargetCount(ctx context.Context, sid int64, event logparse.MsgType) (int64, error) {
	const q = `SELECT count(*) as total FROM server_log s WHERE s.server_id = $1 AND event_type = $2`
	var count int64
	if err := db.c.QueryRow(ctx, q, sid, event).Scan(&count); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player event target count")
	}
	return count, nil
}

func (db *pgStore) getGlobalDamage(ctx context.Context) (int64, error) {
	const q = `SELECT sum(s.damage) as total FROM server_log s WHERE event_type = $1`
	var dmg int64
	if err := db.c.QueryRow(ctx, q, logparse.Damage).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player damage sum")
	}
	return dmg, nil
}

func (db *pgStore) getGlobalMedicDrops(ctx context.Context) (int64, error) {
	const q = `SELECT count(*) as total FROM server_log s WHERE s.event_type = $1`
	var dmg int64
	if err := db.c.QueryRow(ctx, q, logparse.MedicDeath).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch server medic drops sum")
	}
	return dmg, nil
}

func (db *pgStore) getGlobalUses(ctx context.Context) (int64, error) {
	const q = `SELECT count(*) as total FROM server_log s WHERE s.event_type = $1`
	var dmg int64
	if err := db.c.QueryRow(ctx, q, logparse.EmptyUber).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player damage sum")
	}
	return dmg, nil
}

func (db *pgStore) getGlobalHealing(ctx context.Context) (int64, error) {
	const q = `SELECT sum(s.healing) as total FROM server_log s`
	var dmg int64
	if err := db.c.QueryRow(ctx, q).Scan(&dmg); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player healing sum")
	}
	return dmg, nil
}

func (db *pgStore) getGlobalEventCount(ctx context.Context, event logparse.MsgType) (int64, error) {
	const q = `SELECT count(*) as total FROM server_log s WHERE event_type = $1`
	var count int64
	if err := db.c.QueryRow(ctx, q, event).Scan(&count); err != nil {
		return 0, errors.Wrapf(err, "Failed to fetch player event count")
	}
	return count, nil
}
