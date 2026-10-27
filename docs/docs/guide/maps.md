# Map Statistics

The **Map Stats** page (`/maps`, requires the `stats_read` permission) shows how your
community's games break down per map. It uses the same
[stats](stats.md) data as the player leaderboards, aggregated by map.

## Views

- **Summary strip** — totals for the selected window: maps played, matches, decided,
  drawn, playtime and rounds.
- **Map table** — one row per map with matches played, decided, drawn, red/blu wins,
  playtime, rounds, stalemates, players, appearances, kills, damage, healing, captures,
  blocked caps, and first/last played. Every column is sortable.
- **Trend chart** — a time series for the selected metric across the window.
- **Maps by type** — the same metric broken down by map prefix (e.g. `cp_`, `pl_`,
  `koth_`).

## Filters

- **Time range** — last 7 / 30 / 90 days, last year, or all time. The default is the last
  90 days.
- **Metric** — playtime, matches, or decided. Controls the trend chart.
- **Stats group** — the game-mode bucket the stats are aggregated for (e.g. casual, 1ku,
  dodgeball, mge). See [Stats Overview](stats.md).
- **Map name** — type to filter to specific maps.

Clicking a map's **View Matches** link opens the match history filtered to that map.
