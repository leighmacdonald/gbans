package app

import (
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"time"
)

type statType int

const (
	globalStats statType = iota
	//mapStats
	playerStats
	serverStats
)

type serverGameState struct {
	currentMap   string
	roundStarted time.Time
	roundEnded   time.Time
}

// StatTrak tracks stats for server events.
//
// For a simple example of usage, see internal/cmd/stats.go
//
// TODO
// - Use medic death event to calc medic healing count
// - Use healed event for tracking healing received
// - individual game state cache to track who is on winning team
// - Filter out certain pre-game events likes kills/damage
// - Track current player session
// - Track player playtime per class
// - Track server playtime per class
// - Track global playtime per class
// - Track player midfights won
// - Track player biggest killstreaks (min 18 players in server)
// - Track server biggest killstreaks (min 18 players in server)
// - Track global biggest killstreaks (min 18 players in server)
// - Track player classes killed
// - Track player classes killedBy
// - Track server classes killed
// - Track server classes killedBy
// - Track global classes killed
// - Track global classes killedBy
// - Calculate player points
// - Calculate server points
// - Calculate global points
// - Track player weapon stats
// - Track server weapon stats
// - Track global weapon stats
// - calc HealsTaken (live round time only)
// - calc Heals/min (live round time only)
// - calc Dmg/min (live round time only)
// - calc DmgTaken/min (live round time only)
// - Count headshots
// - Count airshots
// - Count headshots
// - Track current map to get correct map stats. Tracking the sm_nextmap cvar may partially work for old data.
//   Update sourcemod plugin to send log event with the current map.
// - Simplify implementation of the maps with generics
// - Track players taking packs when they are close to 100% hp
type StatTrak struct {
	globalAllTimeStats *model.GlobalStats
	serversAlltime     map[int64]*model.ServerStats
	//mapsAlltime    = map[string]*model.MapStats
	playersAlltime map[int64]*model.PlayerStats

	// Stats within date windows
	globalByMonth  map[int]map[time.Month]*model.GlobalStats
	serversByMonth map[int]map[time.Month]map[int64]*model.ServerStats
	playersByMonth map[int]map[time.Month]*model.PlayerStats
	globalByDay    map[int]map[int]*model.GlobalStats
	serversByDay   map[int]map[int]map[int64]*model.ServerStats
	playersByDay   map[int]map[int]*model.PlayerStats
	globalByWeek   map[int]map[int]*model.GlobalStats
	serversByWeek  map[int]map[int]map[int64]*model.ServerStats
	playersByWeek  map[int]map[int]*model.PlayerStats

	// Keeps track of active game stat events for each of the servers
	serverGameStates map[string]*serverGameState
}

func NewStatTrak() StatTrak {
	return StatTrak{
		globalAllTimeStats: &model.GlobalStats{},
		serversAlltime:     map[int64]*model.ServerStats{},
		//mapsAlltime    = map[string]*model.MapStats{},
		playersAlltime: map[int64]*model.PlayerStats{},

		// Stats within date windows
		globalByMonth:  map[int]map[time.Month]*model.GlobalStats{},
		serversByMonth: map[int]map[time.Month]map[int64]*model.ServerStats{},
		playersByMonth: map[int]map[time.Month]*model.PlayerStats{},
		globalByDay:    map[int]map[int]*model.GlobalStats{},
		serversByDay:   map[int]map[int]map[int64]*model.ServerStats{},
		playersByDay:   map[int]map[int]*model.PlayerStats{},
		globalByWeek:   map[int]map[int]*model.GlobalStats{},
		serversByWeek:  map[int]map[int]map[int64]*model.ServerStats{},
		playersByWeek:  map[int]map[int]*model.PlayerStats{},

		// Keeps track of active game stat events for each of the servers
		serverGameStates: map[string]*serverGameState{},
	}
}

func (s *StatTrak) getStatByMonth(st statType, year int, month time.Month, extraId int64) any {
	switch st {
	case globalStats:
		_, found := s.globalByMonth[year]
		if !found {
			s.globalByMonth[year] = map[time.Month]*model.GlobalStats{}
		}
		return s.globalByMonth[year][month]
	case serverStats:
		_, found := s.serversByMonth[year]
		if !found {
			s.serversByMonth[year] = make(map[time.Month]map[int64]*model.ServerStats)
			s.serversByMonth[year][month] = map[int64]*model.ServerStats{}
		}
		y, foundSrv := s.serversByMonth[year][month][extraId]
		if !foundSrv {
			y = &model.ServerStats{}
			s.serversByMonth[year][month][extraId] = y
		}
		return y
	default: // playerStats
		_, found := s.playersByMonth[year]
		if !found {
			s.playersByMonth[year] = map[time.Month]*model.PlayerStats{}
		}
		return s.playersByMonth[year][month]
	}
}

func (s *StatTrak) getStatByWeek(st statType, year int, week int, extraId int64) any {
	switch st {
	case globalStats:
		_, found := s.globalByWeek[year]
		if !found {
			s.globalByWeek[year] = map[int]*model.GlobalStats{}
		}
		return s.globalByWeek[year][week]
	case serverStats:
		_, found := s.serversByWeek[year]
		if !found {
			s.serversByWeek[year] = make(map[int]map[int64]*model.ServerStats)
			s.serversByWeek[year][week] = map[int64]*model.ServerStats{}
		}
		y, foundSrv := s.serversByWeek[year][week][extraId]
		if !foundSrv {
			y = &model.ServerStats{}
			s.serversByWeek[year][week][extraId] = y
		}
		return y
	default: // playerStats
		_, found := s.playersByWeek[year]
		if !found {
			s.playersByDay[year] = map[int]*model.PlayerStats{}
		}
		return s.playersByWeek[year][week]
	}
}

func (s *StatTrak) getStatByDay(st statType, year int, day int, extraId int64) any {
	switch st {
	case globalStats:
		_, found := s.globalByDay[year]
		if !found {
			s.globalByDay[year] = map[int]*model.GlobalStats{}
		}
		return s.globalByDay[year][day]
	case serverStats:
		_, found := s.serversByDay[year]
		if !found {
			s.serversByDay[year] = make(map[int]map[int64]*model.ServerStats)
			s.serversByDay[year][day] = map[int64]*model.ServerStats{}
		}
		y, foundSrv := s.serversByDay[year][day][extraId]
		if !foundSrv {
			y = &model.ServerStats{}
			s.serversByDay[year][day][extraId] = y
		}
		return y
	default: // playerStats
		y, found := s.playersByDay[year]
		if !found {
			y = map[int]*model.PlayerStats{}
			s.playersByDay[year] = y
		}
		return s.playersByDay[year][day]
	}
}

func (s *StatTrak) getStatByAlltime(st statType, extraId int64) any {
	switch st {
	case globalStats:
		return s.globalAllTimeStats
	case serverStats:
		_, found := s.serversAlltime[extraId]
		if !found {
			s.serversAlltime[extraId] = &model.ServerStats{}
		}
		return s.serversAlltime[extraId]
	default: // playerStats
		_, found := s.playersAlltime[extraId]
		if !found {
			s.playersAlltime[extraId] = &model.PlayerStats{}
		}
		return s.playersAlltime[extraId]
	}
}

func (s *StatTrak) Read(event model.ServerEvent) error {
	var (
		year, week = event.CreatedOn.ISOWeek()
		month      = event.CreatedOn.Month()
		day        = event.CreatedOn.YearDay()

		globalAlltime = s.getStatByAlltime(globalStats, 0).(*model.GlobalStats)
		globalMonthly = s.getStatByMonth(globalStats, year, month, 0).(*model.GlobalStats)
		globalDaily   = s.getStatByDay(globalStats, year, day, 0).(*model.GlobalStats)
		globalWeekly  = s.getStatByWeek(globalStats, year, week, 0).(*model.GlobalStats)

		serverAlltime = s.getStatByAlltime(serverStats, event.Server.ServerID).(*model.ServerStats)
		serverMonthly = s.getStatByMonth(serverStats, year, month, event.Server.ServerID).(*model.ServerStats)
		serverDaily   = s.getStatByDay(serverStats, year, day, event.Server.ServerID).(*model.ServerStats)
		serverWeekly  = s.getStatByWeek(serverStats, year, week, event.Server.ServerID).(*model.ServerStats)

		sourceAlltime = s.getStatByAlltime(playerStats, event.Source.SteamID.Int64()).(*model.PlayerStats)
		sourceMonthly = s.getStatByMonth(playerStats, year, month, event.Source.SteamID.Int64()).(*model.PlayerStats)
		sourceDaily   = s.getStatByDay(playerStats, year, day, event.Source.SteamID.Int64()).(*model.PlayerStats)
		sourceWeekly  = s.getStatByWeek(playerStats, year, week, event.Source.SteamID.Int64()).(*model.PlayerStats)

		targetAlltime = s.getStatByAlltime(playerStats, event.Target.SteamID.Int64()).(*model.PlayerStats)
		targetMonthly = s.getStatByMonth(playerStats, year, month, event.Target.SteamID.Int64()).(*model.PlayerStats)
		targetDaily   = s.getStatByDay(playerStats, year, day, event.Target.SteamID.Int64()).(*model.PlayerStats)
		targetWeekly  = s.getStatByWeek(playerStats, year, week, event.Target.SteamID.Int64()).(*model.PlayerStats)
	)
	switch event.EventType {
	case logparse.JoinedTeam:
	// Track game team for wins
	case logparse.KillAssist:
		globalAlltime.Assists++
		globalMonthly.Assists++
		globalDaily.Assists++
		globalWeekly.Assists++

		serverAlltime.Assists++
		serverMonthly.Assists++
		serverDaily.Assists++
		serverWeekly.Assists++

		sourceAlltime.Assists++
		sourceMonthly.Assists++
		sourceDaily.Assists++
		sourceWeekly.Assists++

	case logparse.Healed:
		globalDaily.Healing += event.Healing
		globalWeekly.Healing += event.Healing
		globalMonthly.Healing += event.Healing
		globalAlltime.Healing += event.Healing

		serverDaily.Healing += event.Healing
		serverWeekly.Healing += event.Healing
		serverMonthly.Healing += event.Healing
		serverAlltime.Healing += event.Healing

		sourceDaily.Healing += event.Healing
		sourceWeekly.Healing += event.Healing
		sourceMonthly.Healing += event.Healing
		sourceAlltime.Healing += event.Healing

		targetDaily.HealingTaken += event.Healing
		targetWeekly.HealingTaken += event.Healing
		targetMonthly.HealingTaken += event.Healing
		targetAlltime.HealingTaken += event.Healing

	case logparse.Connected:
	// Add player to game state, remove
	case logparse.Disconnected:
	// remove player from game state
	case logparse.Say:
		globalAlltime.Messages++
		globalMonthly.Messages++
		globalDaily.Messages++
		globalWeekly.Messages++

		serverAlltime.Messages++
		serverMonthly.Messages++
		serverDaily.Messages++
		serverWeekly.Messages++

		sourceAlltime.Messages++
		sourceMonthly.Messages++
		sourceDaily.Messages++
		sourceWeekly.Messages++

	case logparse.SayTeam:
		globalAlltime.MessagesTeam++
		globalMonthly.MessagesTeam++
		globalDaily.MessagesTeam++
		globalWeekly.MessagesTeam++

		serverAlltime.MessagesTeam++
		serverMonthly.MessagesTeam++
		serverDaily.MessagesTeam++
		serverWeekly.MessagesTeam++

		sourceAlltime.MessagesTeam++
		sourceMonthly.MessagesTeam++
		sourceDaily.MessagesTeam++
		sourceWeekly.MessagesTeam++

	case logparse.MedicDeath:
		// Count drops
		// TODO verify the calcs are correct for source & target drops
		uberPct, ok := event.MetaData["uber"].(int64)
		if ok && uberPct >= 100 {
			globalAlltime.MedicDroppedUber++
			globalMonthly.MedicDroppedUber++
			globalDaily.MedicDroppedUber++
			globalWeekly.MedicDroppedUber++

			serverAlltime.MedicDroppedUber++
			serverMonthly.MedicDroppedUber++
			serverDaily.MedicDroppedUber++
			serverWeekly.MedicDroppedUber++

			sourceAlltime.MedicDroppedUber++
			sourceMonthly.MedicDroppedUber++
			sourceDaily.MedicDroppedUber++
			sourceWeekly.MedicDroppedUber++
		}
	case logparse.WTeamFinalScore:
		// Win/loss rates
	case logparse.BuiltObject:
		// bob the builders
		globalAlltime.ObjectBuilt++
		globalMonthly.ObjectBuilt++
		globalDaily.ObjectBuilt++
		globalWeekly.ObjectBuilt++

		serverAlltime.ObjectBuilt++
		serverMonthly.ObjectBuilt++
		serverDaily.ObjectBuilt++
		serverWeekly.ObjectBuilt++

		sourceAlltime.ObjectBuilt++
		sourceMonthly.ObjectBuilt++
		sourceDaily.ObjectBuilt++
		sourceWeekly.ObjectBuilt++

	case logparse.CaptureBlocked:
		// Blocks
		globalAlltime.PointDefends++
		globalMonthly.PointDefends++
		globalDaily.PointDefends++
		globalWeekly.PointDefends++

		serverAlltime.PointDefends++
		serverMonthly.PointDefends++
		serverDaily.PointDefends++
		serverWeekly.PointDefends++

		sourceAlltime.PointDefends++
		sourceMonthly.PointDefends++
		sourceDaily.PointDefends++
		sourceWeekly.PointDefends++

	case logparse.PointCaptured:
		// captures, multiple people
		// TODO calc all people who capped in meta_data
		globalAlltime.PointCaptures++
		globalMonthly.PointCaptures++
		globalDaily.PointCaptures++
		globalWeekly.PointCaptures++

		serverAlltime.PointCaptures++
		serverMonthly.PointCaptures++
		serverDaily.PointCaptures++
		serverWeekly.PointCaptures++

		sourceAlltime.PointCaptures++
		sourceMonthly.PointCaptures++
		sourceDaily.PointCaptures++
		sourceWeekly.PointCaptures++

	case logparse.Domination:
		globalAlltime.Dominations++
		globalMonthly.Dominations++
		globalDaily.Dominations++
		globalWeekly.Dominations++

		serverAlltime.Dominations++
		serverMonthly.Dominations++
		serverDaily.Dominations++
		serverWeekly.Dominations++

		sourceAlltime.Dominations++
		sourceMonthly.Dominations++
		sourceDaily.Dominations++
		sourceWeekly.Dominations++

		targetAlltime.Dominated++
		targetMonthly.Dominated++
		targetDaily.Dominated++
		targetWeekly.Dominated++

	case logparse.Revenge:
		globalAlltime.Revenges++
		globalMonthly.Revenges++
		globalDaily.Revenges++
		globalWeekly.Revenges++

		serverAlltime.Revenges++
		serverMonthly.Revenges++
		serverDaily.Revenges++
		serverWeekly.Revenges++

		sourceAlltime.Revenges++
		sourceMonthly.Revenges++
		sourceDaily.Revenges++
		sourceWeekly.Revenges++

	case logparse.Suicide:
		globalAlltime.Suicides++
		globalMonthly.Suicides++
		globalDaily.Suicides++
		globalWeekly.Suicides++

		serverAlltime.Suicides++
		serverMonthly.Suicides++
		serverDaily.Suicides++
		serverWeekly.Suicides++

		sourceAlltime.Suicides++
		sourceMonthly.Suicides++
		sourceDaily.Suicides++
		sourceWeekly.Suicides++

	case logparse.WRoundWin:
	case logparse.WRoundLen:
	case logparse.Extinguished:
		globalAlltime.Extinguishes++
		globalMonthly.Extinguishes++
		globalDaily.Extinguishes++
		globalWeekly.Extinguishes++

		serverAlltime.Extinguishes++
		serverMonthly.Extinguishes++
		serverDaily.Extinguishes++
		serverWeekly.Extinguishes++

		sourceAlltime.Extinguishes++
		sourceMonthly.Extinguishes++
		sourceDaily.Extinguishes++
		sourceWeekly.Extinguishes++

	case logparse.SpawnedAs:
		switch event.PlayerClass {
		case logparse.Scout:
			globalAlltime.SpawnScout++
			globalMonthly.SpawnScout++
			globalDaily.SpawnScout++
			globalWeekly.SpawnScout++

			serverAlltime.SpawnScout++
			serverMonthly.SpawnScout++
			serverDaily.SpawnScout++
			serverWeekly.SpawnScout++

			sourceAlltime.SpawnScout++
			sourceMonthly.SpawnScout++
			sourceDaily.SpawnScout++
			sourceWeekly.SpawnScout++

		case logparse.Soldier:
			globalAlltime.SpawnSoldier++
			globalMonthly.SpawnSoldier++
			globalDaily.SpawnSoldier++
			globalWeekly.SpawnSoldier++

			serverAlltime.SpawnSoldier++
			serverMonthly.SpawnSoldier++
			serverDaily.SpawnSoldier++
			serverWeekly.SpawnSoldier++

			sourceAlltime.SpawnSoldier++
			sourceMonthly.SpawnSoldier++
			sourceDaily.SpawnSoldier++
			sourceWeekly.SpawnSoldier++

		case logparse.Pyro:
			globalAlltime.SpawnPyro++
			globalMonthly.SpawnPyro++
			globalDaily.SpawnPyro++
			globalWeekly.SpawnPyro++

			serverAlltime.SpawnPyro++
			serverMonthly.SpawnPyro++
			serverDaily.SpawnPyro++
			serverWeekly.SpawnPyro++

			sourceAlltime.SpawnPyro++
			sourceMonthly.SpawnPyro++
			sourceDaily.SpawnPyro++
			sourceWeekly.SpawnPyro++

		case logparse.Demo:
			globalAlltime.SpawnDemo++
			globalMonthly.SpawnDemo++
			globalDaily.SpawnDemo++
			globalWeekly.SpawnDemo++

			serverAlltime.SpawnDemo++
			serverMonthly.SpawnDemo++
			serverDaily.SpawnDemo++
			serverWeekly.SpawnDemo++

			sourceAlltime.SpawnDemo++
			sourceMonthly.SpawnDemo++
			sourceDaily.SpawnDemo++
			sourceWeekly.SpawnDemo++

		case logparse.Heavy:
			globalAlltime.SpawnHeavy++
			globalMonthly.SpawnHeavy++
			globalDaily.SpawnHeavy++
			globalWeekly.SpawnHeavy++

			serverAlltime.SpawnHeavy++
			serverMonthly.SpawnHeavy++
			serverDaily.SpawnHeavy++
			serverWeekly.SpawnHeavy++

			sourceAlltime.SpawnHeavy++
			sourceMonthly.SpawnHeavy++
			sourceDaily.SpawnHeavy++
			sourceWeekly.SpawnHeavy++

		case logparse.Engineer:
			globalAlltime.SpawnEngineer++
			globalMonthly.SpawnEngineer++
			globalDaily.SpawnEngineer++
			globalWeekly.SpawnEngineer++

			serverAlltime.SpawnEngineer++
			serverMonthly.SpawnEngineer++
			serverDaily.SpawnEngineer++
			serverWeekly.SpawnEngineer++

			sourceAlltime.SpawnEngineer++
			sourceMonthly.SpawnEngineer++
			sourceDaily.SpawnEngineer++
			sourceWeekly.SpawnEngineer++
		case logparse.Medic:
			globalAlltime.SpawnMedic++
			globalMonthly.SpawnMedic++
			globalDaily.SpawnMedic++
			globalWeekly.SpawnMedic++

			serverAlltime.SpawnMedic++
			serverMonthly.SpawnMedic++
			serverDaily.SpawnMedic++
			serverWeekly.SpawnMedic++

			sourceAlltime.SpawnMedic++
			sourceMonthly.SpawnMedic++
			sourceDaily.SpawnMedic++
			sourceWeekly.SpawnMedic++
		case logparse.Sniper:
			globalAlltime.SpawnSniper++
			globalMonthly.SpawnSniper++
			globalDaily.SpawnSniper++
			globalWeekly.SpawnSniper++

			serverAlltime.SpawnSniper++
			serverMonthly.SpawnSniper++
			serverDaily.SpawnSniper++
			serverWeekly.SpawnSniper++

			sourceAlltime.SpawnSniper++
			sourceMonthly.SpawnSniper++
			sourceDaily.SpawnSniper++
			sourceWeekly.SpawnSniper++

		case logparse.Spy:
			globalAlltime.SpawnSpy++
			globalMonthly.SpawnSpy++
			globalDaily.SpawnSpy++
			globalWeekly.SpawnSpy++

			serverAlltime.SpawnSpy++
			serverMonthly.SpawnSpy++
			serverDaily.SpawnSpy++
			serverWeekly.SpawnSpy++

			sourceAlltime.SpawnSpy++
			sourceMonthly.SpawnSpy++
			sourceDaily.SpawnSpy++
			sourceWeekly.SpawnSpy++
		}
	case logparse.Pickup:
		switch event.Item {
		case logparse.ItemAmmoLarge:
			globalAlltime.PickupAmmoLarge++
			globalMonthly.PickupAmmoLarge++
			globalDaily.PickupAmmoLarge++
			globalWeekly.PickupAmmoLarge++

			serverAlltime.PickupAmmoLarge++
			serverMonthly.PickupAmmoLarge++
			serverDaily.PickupAmmoLarge++
			serverWeekly.PickupAmmoLarge++

			sourceAlltime.PickupAmmoLarge++
			sourceMonthly.PickupAmmoLarge++
			sourceDaily.PickupAmmoLarge++
			sourceWeekly.PickupAmmoLarge++

		case logparse.ItemAmmoMedium:
			globalAlltime.PickupAmmoMedium++
			globalMonthly.PickupAmmoMedium++
			globalDaily.PickupAmmoMedium++
			globalWeekly.PickupAmmoMedium++

			serverAlltime.PickupAmmoMedium++
			serverMonthly.PickupAmmoMedium++
			serverDaily.PickupAmmoMedium++
			serverWeekly.PickupAmmoMedium++

			sourceAlltime.PickupAmmoMedium++
			sourceMonthly.PickupAmmoMedium++
			sourceDaily.PickupAmmoMedium++
			sourceWeekly.PickupAmmoMedium++

		case logparse.ItemAmmoSmall:
			globalAlltime.PickupAmmoSmall++
			globalMonthly.PickupAmmoSmall++
			globalDaily.PickupAmmoSmall++
			globalWeekly.PickupAmmoSmall++

			serverAlltime.PickupAmmoSmall++
			serverMonthly.PickupAmmoSmall++
			serverDaily.PickupAmmoSmall++
			serverWeekly.PickupAmmoSmall++

			sourceAlltime.PickupAmmoSmall++
			sourceMonthly.PickupAmmoSmall++
			sourceDaily.PickupAmmoSmall++
			sourceWeekly.PickupAmmoSmall++

		case logparse.ItemHPLarge:
			globalAlltime.PickupHPLarge++
			globalMonthly.PickupHPLarge++
			globalDaily.PickupHPLarge++
			globalWeekly.PickupHPLarge++

			serverAlltime.PickupHPLarge++
			serverMonthly.PickupHPLarge++
			serverDaily.PickupHPLarge++
			serverWeekly.PickupHPLarge++

			sourceAlltime.PickupHPLarge++
			sourceMonthly.PickupHPLarge++
			sourceDaily.PickupHPLarge++
			sourceWeekly.PickupHPLarge++

		case logparse.ItemHPMedium:
			globalAlltime.PickupHPMedium++
			globalMonthly.PickupHPMedium++
			globalDaily.PickupHPMedium++
			globalWeekly.PickupHPMedium++

			serverAlltime.PickupHPMedium++
			serverMonthly.PickupHPMedium++
			serverDaily.PickupHPMedium++
			serverWeekly.PickupHPMedium++

			sourceAlltime.PickupHPMedium++
			sourceMonthly.PickupHPMedium++
			sourceDaily.PickupHPMedium++
			sourceWeekly.PickupHPMedium++

		case logparse.ItemHPSmall:
			globalAlltime.PickupHPSmall++
			globalMonthly.PickupHPSmall++
			globalDaily.PickupHPSmall++
			globalWeekly.PickupHPSmall++

			serverAlltime.PickupHPSmall++
			serverMonthly.PickupHPSmall++
			serverDaily.PickupHPSmall++
			serverWeekly.PickupHPSmall++

			sourceAlltime.PickupHPSmall++
			sourceMonthly.PickupHPSmall++
			sourceDaily.PickupHPSmall++
			sourceWeekly.PickupHPSmall++

		}
	case logparse.ShotFired:
		globalAlltime.Shots++
		globalMonthly.Shots++
		globalDaily.Shots++
		globalWeekly.Shots++

		serverAlltime.Shots++
		serverMonthly.Shots++
		serverDaily.Shots++
		serverWeekly.Shots++

		sourceAlltime.Shots++
		sourceMonthly.Shots++
		sourceDaily.Shots++
		sourceWeekly.Shots++

	case logparse.ShotHit:
		globalAlltime.Hits++
		globalMonthly.Hits++
		globalDaily.Hits++
		globalWeekly.Hits++

		serverAlltime.Hits++
		serverMonthly.Hits++
		serverDaily.Hits++
		serverWeekly.Hits++

		sourceAlltime.Hits++
		sourceMonthly.Hits++
		sourceDaily.Hits++
		sourceWeekly.Hits++

	case logparse.Killed:
		globalAlltime.Kills++
		globalMonthly.Kills++
		globalDaily.Kills++
		globalWeekly.Kills++

		serverAlltime.Kills++
		serverMonthly.Kills++
		serverDaily.Kills++
		serverWeekly.Kills++

		sourceAlltime.Kills++
		sourceMonthly.Kills++
		sourceDaily.Kills++
		sourceWeekly.Kills++

		targetAlltime.Deaths++
		targetMonthly.Deaths++
		targetDaily.Deaths++
		targetWeekly.Deaths++

	case logparse.Damage:
		globalAlltime.Damage += event.Damage
		globalMonthly.Damage += event.Damage
		globalDaily.Damage += event.Damage
		globalWeekly.Damage += event.Damage

		serverAlltime.Damage += event.Damage
		serverMonthly.Damage += event.Damage
		serverDaily.Damage += event.Damage
		serverWeekly.Damage += event.Damage

		sourceAlltime.Damage += event.Damage
		sourceMonthly.Damage += event.Damage
		sourceDaily.Damage += event.Damage
		sourceWeekly.Damage += event.Damage

		targetAlltime.DamageTaken += event.Damage
		targetMonthly.DamageTaken += event.Damage
		targetDaily.DamageTaken += event.Damage
		targetWeekly.DamageTaken += event.Damage
	}

	return nil
}
