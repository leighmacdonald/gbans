package anticheat

import (
	"context"
	"errors"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func NewAntiCheatRepository(database database.Database) *anticheatRepository {
	return &anticheatRepository{db: database}
}

type anticheatRepository struct {
	db database.Database
}

func (a anticheatRepository) Query(ctx context.Context, query AnticheatQuery) ([]AnticheatEntry, error) {
	builder := a.db.Builder().
		Select("a.anticheat_id", "a.steam_id", "a.name", "a.detection", "a.summary", "a.demo_id", "a.demo_tick",
			"a.server_id", "a.raw_log", "s.short_name", "p.personaname", "p.avatarhash", "a.created_on",
			"RANK () OVER (PARTITION BY a.steam_id, a.detection ORDER BY a.created_on) triggered").
		From("anticheat a").
		LeftJoin("server s USING(server_id)").
		LeftJoin("person p USING(steam_id)")

	var filters sq.And
	if query.Summary != "" {
		filters = append(filters, sq.Like{"lower(a.summary)": "%" + strings.ToLower(query.Summary) + "%"})
	}
	if query.SteamID != "" {
		sid := steamid.New(query.SteamID)
		filters = append(filters, sq.Eq{"a.steam_id": sid.Int64()})
	}
	if query.ServerID > 0 {
		filters = append(filters, sq.Eq{"a.server_id": query.ServerID})
	}
	if query.Name != "" {
		filters = append(filters, sq.Like{"a.name": "%" + strings.ToLower(query.Name) + "%"})
	}
	if query.Detection != "" && string(query.Detection) != "any" {
		filters = append(filters, sq.Eq{"a.detection": query.Detection})
	}

	if len(filters) > 0 {
		builder = builder.Where(filters)
	}

	builder = query.ApplySafeOrder(builder, map[string][]string{
		"p.": {"personaname", "avatarhash"},
		"a.": {"anticheat_id", "steam_id", "name", "detection", "summary", "demo_id", "demo_tick", "server_id", "raw_log", "created_on"},
		"s.": {"short_name"},
	}, "a.created_on")

	builder = query.ApplyLimitOffsetDefault(builder)

	rows, errRows := a.db.QueryBuilder(ctx, nil, builder)
	if errRows != nil {
		return nil, errRows
	}

	defer rows.Close()

	var entries []AnticheatEntry

	for rows.Next() {
		var entry AnticheatEntry
		if err := rows.Scan(&entry.AnticheatID, &entry.SteamID, &entry.Personaname, &entry.Detection, &entry.Summary,
			&entry.DemoID, &entry.DemoTick, &entry.ServerID, &entry.RawLog, &entry.ServerName, &entry.Personaname,
			&entry.AvatarHash, &entry.CreatedOn, &entry.Triggered); err != nil {
			return nil, a.db.DBErr(err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (a anticheatRepository) DetectionsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]logparse.StacEntry, error) {
	rows, errRows := a.db.QueryBuilder(ctx, nil, a.db.Builder().
		Select("anticheat_id", "steam_id", "name", "detection", "summary", "demo_id", "server_id", "raw_log", "s.short_name").
		From("anticheat").
		LeftJoin("server s USING(server_id)").
		Where(sq.Eq{"steam_id": steamID.Int64()}))
	if errRows != nil {
		return nil, errRows
	}

	defer rows.Close()

	var entries []logparse.StacEntry

	for rows.Next() {
		var entry logparse.StacEntry
		if err := rows.Scan(&entry.AnticheatID); err != nil {
			return nil, a.db.DBErr(err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (a anticheatRepository) DetectionsByType(ctx context.Context, detectionType logparse.Detection) ([]logparse.StacEntry, error) {
	rows, errRows := a.db.QueryBuilder(ctx, nil, a.db.Builder().
		Select("anticheat_id", "steam_id", "name", "detection", "summary", "demo_id", "server_id", "raw_log", "s.short_name").
		From("anticheat").
		LeftJoin("server s USING(server_id)").
		Where(sq.Eq{"detection": detectionType}))
	if errRows != nil {
		return nil, errRows
	}

	defer rows.Close()

	var entries []logparse.StacEntry

	for rows.Next() {
		var entry logparse.StacEntry
		if err := rows.Scan(&entry.AnticheatID); err != nil {
			return nil, a.db.DBErr(err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (a anticheatRepository) SaveEntries(ctx context.Context, entries []logparse.StacEntry) error {
	for _, entry := range entries {
		if err := a.db.ExecInsertBuilder(ctx, nil, a.db.Builder().
			Insert("anticheat").
			SetMap(map[string]interface{}{
				"steam_id":   entry.SteamID.Int64(),
				"name":       entry.Name,
				"detection":  entry.Detection,
				"summary":    entry.Summary,
				"demo_id":    entry.DemoID,
				"server_id":  entry.ServerID,
				"raw_log":    entry.RawLog,
				"created_on": entry.CreatedOn.Truncate(time.Second),
			})); err != nil && !errors.Is(err, database.ErrDuplicate) {
			return a.db.DBErr(err)
		}
	}

	return nil
}

type demoIDMap struct {
	demoID      int
	anticheatID int
	title       string
}

func (a anticheatRepository) getMissingIDMap(ctx context.Context, limit uint64) ([]demoIDMap, []string, error) {
	// Find entries with no demo_id attached, but which also are marked with a specific demo
	rows, errRows := a.db.QueryBuilder(ctx, nil, a.db.Builder().
		Select("anticheat_id", "demo_name").
		From("anticheat").
		Where(sq.And{sq.Eq{"demo_id": 0}, sq.NotEq{"demo_name": ""}}).
		OrderBy("created_on desc").
		Limit(limit))
	if errRows != nil {
		return nil, nil, errRows
	}

	defer rows.Close()

	var ids []demoIDMap
	var titles []string
	for rows.Next() {
		var idm demoIDMap
		if err := rows.Scan(&idm.demoID, &idm.title); err != nil {
			return nil, nil, a.db.DBErr(err)
		}

		titles = append(titles, idm.title)
		ids = append(ids, idm)
	}

	return ids, titles, nil
}

func (a anticheatRepository) getDemoIDsByTitle(ctx context.Context, titles []string) ([]demoIDMap, error) {
	demos, errDemos := a.db.QueryBuilder(ctx, nil, a.db.Builder().
		Select("demo_id", "title").
		From("demo").
		Where(sq.Eq{"title": titles}))
	if errDemos != nil {
		return nil, errDemos
	}

	defer demos.Close()

	var demoMaps []demoIDMap

	for demos.Next() {
		var m demoIDMap
		if errScan := demos.Scan(&m.demoID, &m.title); errScan != nil {
			return nil, a.db.DBErr(errScan)
		}

		demoMaps = append(demoMaps, m)
	}

	return demoMaps, nil
}

func (a anticheatRepository) updateTitleMapping(ctx context.Context, titleMap []demoIDMap) error {
	for _, tm := range titleMap {
		if errExec := a.db.ExecUpdateBuilder(ctx, nil, a.db.Builder().
			Update("anticheat").
			Set("demo_id", tm.demoID).
			Where("anticheat_id", tm.anticheatID)); errExec != nil {
			return errExec
		}
	}

	return nil
}

// SyncDemoIDs is used to try and attach known demo_ids to an anticheat record. This is done like this because
// we don't neccessarilly know what the demo_id will be ahead of time.
func (a anticheatRepository) SyncDemoIDs(ctx context.Context, limit uint64) error {
	ids, titles, err := a.getMissingIDMap(ctx, limit)
	if err != nil {
		return err
	}

	titleMap, errIDs := a.getDemoIDsByTitle(ctx, titles)
	if errIDs != nil {
		return errIDs
	}

	// Assing associated cheating incident to the demo.
	for i := range ids {
		for _, entry := range ids {
			if titleMap[i].title == entry.title {
				titleMap[i].anticheatID = entry.anticheatID

				break
			}
		}
	}

	return a.updateTitleMapping(ctx, titleMap)
}
