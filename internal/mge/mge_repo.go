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

}
