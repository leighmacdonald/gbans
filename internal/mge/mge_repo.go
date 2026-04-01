package mge

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type PlayerStats struct {
	StatsID     int             `json:"stats_id"`
	Rating      int             `json:"rating"`
	SteamID     steamid.SteamID `json:"steam_id"`
	Personaname string          `json:"personaname"`
	Avatarhash  string          `json:"avatarhash"`
	Name        string          `json:"name"`
	Wins        int             `json:"wins"`
	Losses      int             `json:"losses"`
<<<<<<< HEAD
	LastPlayed  time.Time       `json:"last_played"`
	HitBlip     int             `json:"hit_blip"`
}

type QueryOpts struct {
	query.Filter

	SteamID string `schema:"steam_id"`
}

type Repository struct {
	database.Database
}

func NewRepository(db database.Database) Repository {
	return Repository{Database: db}
}

func (r Repository) Query(ctx context.Context, opts QueryOpts) ([]PlayerStats, int64, error) {
	builder := r.Builder().
		Select("s.stats_id", "s.rating", "s.steamid", "s.name", "s.wins", "s.losses", "to_timestamp(s.lastplayed)", "s.hitblip",
			"coalesce(p.personaname, s.name)", "coalesce(p.avatarhash, '')").
		From("mgemod_stats s").
		LeftJoin("person p ON s.steamid = p.steam_id")
	builder = opts.ApplySafeOrder(opts.ApplyLimitOffsetDefault(builder), map[string][]string{
		"s.": {"rating", "name", "wins", "losses", "lastplayed", "hitblip"},
	}, "s.rating")

	var constraints sq.And
	if opts.SteamID != "" {
		sid := steamid.New(opts.SteamID)
		constraints = append(constraints, sq.Eq{"s.steamid": sid.Int64()})
	}

	rows, err := r.QueryBuilder(ctx, builder.Where(constraints))
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	stats := []PlayerStats{}
	for rows.Next() {
		var stat PlayerStats
		if err := rows.Scan(&stat.StatsID, &stat.Rating, &stat.SteamID, &stat.Name, &stat.Wins,
			&stat.Losses, &stat.LastPlayed, &stat.HitBlip, &stat.Personaname, &stat.Avatarhash); err != nil {
			return nil, 0, database.Err(err)
		}
		stats = append(stats, stat)
	}

	if len(stats) == 0 {
		return stats, 0, nil
	}

	count, errCount := r.GetCount(ctx, r.Builder().
		Select("count(s.stats_id) as count").
		From("mgemod_stats s").
		LeftJoin("person p ON s.steamid = p.steam_id").
		Where(constraints))
	if errCount != nil {
		return nil, 0, database.Err(errCount)
	}

	return stats, count, nil
}

type DuelMode = int

const (
	OneVsOne DuelMode = iota
	TwoVsTwo
)

type Duels struct {
	DuelID             int             `json:"duel_id"`
	Winner             steamid.SteamID `json:"winner"`
	WinnerAvatarhash   string          `json:"winner_avatarhash"`
	WinnerPersonaname  string          `json:"winner_personaname"`
	Winner2            steamid.SteamID `json:"winner2"`
	Winner2Avatarhash  string          `json:"winner2_avatarhash"`
	Winner2Personaname string          `json:"winner2_personaname"`
	Loser              steamid.SteamID `json:"loser"`
	LoserAvatarhash    string          `json:"loser_avatarhash"`
	LoserPersonaname   string          `json:"loser_personaname"`
	Loser2             steamid.SteamID `json:"loser2"`
	Loser2Avatarhash   string          `json:"loser2_avatarhash"`
	Loser2Personaname  string          `json:"loser2_personaname"`
	WinnerScore        int             `json:"winner_score"`
	LoserScore         int             `json:"loser_score"`
	Winlimit           int             `json:"winlimit"`
	GameTime           time.Time       `json:"game_time"`
	MapName            string          `json:"map_name"`
	ArenaName          string          `json:"arena_name"`
}

type HistoryOpts struct {
	query.Filter

	Mode    DuelMode `schema:"mode"`
	Winner  string   `schema:"winner"`
	Loser   string   `schema:"loser"`
	Winner2 string   `schema:"winner2"`
	Loser2  string   `schema:"loser2"`
}

func (r Repository) History(ctx context.Context, opts HistoryOpts) ([]Duels, int64, error) {
	columns := []string{
		"m.winner", "coalesce(w.avatarhash, '')", "coalesce(w.personaname, m.winner::text)",
		"m.loser", "coalesce(l.avatarhash, '')", "coalesce(l.personaname, m.loser::text)",
		"m.winnerscore", "m.loserscore", "m.winlimit",
		"to_timestamp(m.gametime)", "m.mapname", "m.arenaname",
	}
	fromTable := "mgemod_duels m"
	if opts.Mode == TwoVsTwo {
		columns = append(columns,
			"m.winner2", "coalesce(w2.avatarhash, '')", "coalesce(w2.personaname, m.winner2::text)",
			"m.loser2", "coalesce(l2.avatarhash, '')", "coalesce(l2.personaname, m.loser2::text)", "m.duel2_id")
		fromTable = "mgemod_duels_2v2 m"
	} else {
		columns = append(columns, "m.duel_id")
	}

	var constraints sq.Or

	builder := r.Builder().
		Select(columns...).
		From(fromTable).
		LeftJoin("person w ON m.winner = w.steam_id").
		LeftJoin("person l ON m.loser = l.steam_id")

	var ids steamid.Collection
	if opts.Winner != "" {
		ids = append(ids, steamid.New(opts.Winner))
	}
	if opts.Loser != "" {
		ids = append(ids, steamid.New(opts.Loser))
	}
	if opts.Mode == TwoVsTwo {
		if opts.Winner2 != "" {
			ids = append(ids, steamid.New(opts.Winner2))
		}
		if opts.Loser2 != "" {
			ids = append(ids, steamid.New(opts.Loser2))
		}
	}

	if len(ids) > 0 {
		constraints = append(constraints, sq.Eq{"m.winner": ids.ToInt64Slice()}, sq.Eq{"m.loser": ids.ToInt64Slice()})
	}

	if opts.Mode == TwoVsTwo {
		builder = builder.
			LeftJoin("person w2 ON m.winner2 = w2.steam_id").
			LeftJoin("person l2 ON m.loser2 = l2.steam_id")
		if len(ids) > 0 {
			constraints = append(constraints, sq.Eq{"m.winner2": ids.ToInt64Slice()}, sq.Eq{"m.loser2": ids.ToInt64Slice()})
		}
	}

	rows, errRows := r.QueryBuilder(ctx, builder.Where(constraints))
	if errRows != nil {
		return nil, 0, database.Err(errRows)
	}
	defer rows.Close()

	var duels []Duels
	for rows.Next() {
		var duel Duels
		rcvr := []any{
			&duel.Winner, &duel.WinnerAvatarhash, &duel.WinnerPersonaname,
			&duel.Loser, &duel.LoserAvatarhash, &duel.LoserPersonaname,
			&duel.WinnerScore, &duel.LoserScore,
			&duel.Winlimit, &duel.GameTime, &duel.MapName, &duel.ArenaName,
		}
		if opts.Mode == TwoVsTwo {
			rcvr = append(rcvr,
				&duel.Winner2, &duel.Winner2Avatarhash, &duel.Winner2Personaname,
				&duel.Loser2, &duel.Loser2Avatarhash, &duel.Loser2Personaname)
		}
		rcvr = append(rcvr, &duel.DuelID)

		if err := rows.Scan(rcvr...); err != nil {
			return nil, 0, database.Err(err)
		}
		duels = append(duels, duel)
	}

	countBuilder := r.Builder().
		Select("count(*)").
		From(fromTable).LeftJoin("person w ON m.winner = w.steam_id").
		LeftJoin("person l ON m.loser = l.steam_id")

	if opts.Mode == TwoVsTwo {
		countBuilder = countBuilder.
			LeftJoin("person w2 ON m.winner2 = w2.steam_id").
			LeftJoin("person l2 ON m.loser2 = l2.steam_id")
	}

	count, errCount := r.GetCount(ctx, countBuilder.Where(constraints))
	if errCount != nil {
		return nil, 0, database.Err(errCount)
	}

	return duels, count, nil
||||||| parent of 179f35e8 (Add overall ranking table)
=======
	LastPlayed  time.Time       `json:"lastplayed"`
	HotBlip     int             `json:"hitblip"`
}

type QueryOpts struct {
	query.Filter

	SteamID string `schema:"steam_id"`
}

type Repository struct {
	database.Database
}

func NewRepository(db database.Database) Repository {
	return Repository{Database: db}
}

func (r *Repository) Query(ctx context.Context, opts QueryOpts) ([]PlayerStats, int64, error) {
	builder := r.Builder().
		Select("s.stats_id", "s.rating", "s.steamid", "s.name", "s.wins", "s.losses", "to_timestamp(s.lastplayed)", "s.hitblip",
			"coalesce(p.personaname, s.name)", "coalesce(p.avatarhash, '')").
		From("mgemod_stats s").
		LeftJoin("person p ON s.steamid = p.steam_id")
	builder = opts.ApplySafeOrder(opts.ApplyLimitOffsetDefault(builder), map[string][]string{
		"s.": {"rating", "name", "wins", "losses", "lastplayed", "hitblip"},
	}, "s.rating")

	var constraints sq.And
	if opts.SteamID != "" {
		sid := steamid.New(opts.SteamID)
		constraints = append(constraints, sq.Eq{"s.steamid": sid.Int64()})
	}

	rows, err := r.QueryBuilder(ctx, builder.Where(constraints))
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	stats := []PlayerStats{}
	for rows.Next() {
		var stat PlayerStats
		if err := rows.Scan(&stat.StatsID, &stat.Rating, &stat.SteamID, &stat.Name, &stat.Wins,
			&stat.Losses, &stat.LastPlayed, &stat.HotBlip, &stat.Personaname, &stat.Avatarhash); err != nil {
			return nil, 0, database.Err(err)
		}
		stats = append(stats, stat)
	}

	if len(stats) == 0 {
		return stats, 0, nil
	}

	count, errCount := r.GetCount(ctx, r.Builder().
		Select("count(s.stats_id) as count").
		From("mgemod_stats s").
		LeftJoin("person p ON s.steamid = p.steam_id").
		Where(constraints))
	if errCount != nil {
		return nil, 0, database.Err(errCount)
	}
	return stats, count, nil
}

func (r *Repository) Duels(ctx context.Context, opts QueryOpts) {

>>>>>>> 179f35e8 (Add overall ranking table)
}
