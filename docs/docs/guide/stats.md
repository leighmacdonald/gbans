# Stats Overview

Stats are generated for many game events. They are summarized into atomic matches and stored in the database. The
design goal is to have a cross between [logs.tf](https://logs.tf)
and [hlstatsx:ce](https://github.com/A1mDev/hlstatsx-community-edition).

## Game Stats page

The `/stats` page (requires the `stats_read` permission) shows leaderboards per stats bucket for a chosen time
window (day, week, month, year, or all time), overall or broken down by kills, healing, weapon, or class.

Leaderboards only include players that meet a minimum activity threshold: a player needs at least 5 kills to
appear on any view except healing, which requires at least 1000 healing. Ranks are still resolved against the
full window, so a player below the threshold has no rank on that leaderboard.

When the signed-in player has stats in the selected window, their rank is shown in the page header. Clicking it
jumps the leaderboard table to the page containing their row and briefly highlights it.

Skill ratings, which are separate from these leaderboards and used for matchmaking, are described in
[Player ratings](#player-ratings).

### Find Player

The filter form has a **Find Player** field that accepts any form of Steam ID or profile link. Submitting the form
locates that player in the selected stats window: the table jumps to the page containing their row, which is
briefly highlighted and stays emphasized for as long as the search is active. The player's full-window rank is
shown in the page header next to your own; clicking it re-runs the jump.

The search is not part of the page URL — it only affects the current page until you submit the form again or
clear the field. If the player has no stats in the selected window, an error is shown instead. Changing the
stats group, time window, or stat type re-runs the search against the new window.

## Stat Buckets

Buckets group matches by game mode (for example `casual` and `1ku`) so their statistics are not skewed by mixing
modes. They are managed from the **Stats** tab on the admin system settings page.

Admins can:

- **Create** a bucket by giving it a unique name (up to 64 characters). New buckets are enabled by default.
- **Delete** a bucket. Deletion is blocked while the bucket still has matches or servers assigned to it, and the
  default bucket can never be deleted.
- **Enable/disable** a bucket. Disabled buckets are hidden from selection for new matches, but their historical
  statistics are kept. The default bucket cannot be disabled.

Buckets can only be modified by users with the `config_write` permission; viewing them only requires `stats_read`.

## Player ratings

Every player has a skill rating per stats bucket, intended for skill-based matchmaking and team balancing. Ratings
are inferred purely from match win/loss: kills, damage and other performance stats do not influence them.

- A new player starts at **1000**.
- A win moves the rating up, a loss moves it down, and a draw counts as a result: it mostly reduces the uncertainty
  of both teams.
- Ratings are **provisional** for the first 5 rated matches. Until then the player reports the default rating and is
  unranked, while the underlying estimate already updates so the real rating is available as soon as they qualify.
- The public rating is produced by the configured calculator. The default TrueSkill calculator maps the underlying
  mean onto a 1000-centred scale (`1000 + 40 * (mu - 25)`); `rating_sigma` and `rating_conservative` (the lower
  confidence bound, `mu - 3 * sigma`) express how confident the estimate is. Elo instead reports its scalar rating
  directly and has no uncertainty.
- Matches without a recorded result (`0-0`) and players who never joined a team are not rated. Players whose recorded
  stats are entirely empty for a match (for example spectators) are also skipped.
- Team membership is taken from the round-end rosters captured in the demo.

### Choosing a rating algorithm

The calculation is pluggable through the `RatingCalculator` interface. Choose an implementation from the
**Rating Algorithm** selector in the admin **Stats** settings tab:

- `trueskill` (default): two-team TrueSkill/Bayesian updates with draw support and per-match skill drift.
- `elo`: classic team Elo starting at 1000 with a K-factor of 32. Fewer moving parts, but no uncertainty estimate.
- `synelo`: team Elo with a dynamic K-factor derived from each player's convergence over their last 500 rated events,
  a Gaussian compression term, a rating floor of 100 and a 50-round shadow period during which the rating stays
  hidden. This is the algorithm described in [issue #696](https://github.com/leighmacdonald/gbans/issues/696).

Changing the algorithm **requires a restart** because the rating service is constructed at startup. The next stats
refresh rebuilds every rating with the new algorithm, so switching rebases all players; existing rating history is
replaced as part of the same replay. Each calculator owns its display scale and its visibility/staleness policy,
which are persisted with the rating; the leaderboard view contains no algorithm-specific rules.

Calculators that need per-player history (such as SynElo) receive each player's identity, rating and a bounded window
of past ratings, expected scores and outcomes. Those expectations and the round count are stored in the rating
history alongside the usual result.

Ratings are rebuilt from scratch on every stats refresh by replaying every rated match in chronological order, so
late-imported demos and deleted matches are handled automatically. The current rating is exposed by
`StatsService.PlayerRating`, rating-over-time snapshots by `StatsService.PlayerRatingHistory`, and the per-bucket
ranking by `StatsService.RatingLeaderboard`. All three require the `stats_read` permission.

### Team balancing

Ratings are also used to split players into balanced teams. The **Balancer** selector in the admin **Stats** settings
tab chooses the implementation; changing it requires a restart.

- `alternating` (default): sorts players by rating and assigns them red/blue/red/blue from strongest to weakest. Fast
  and predictable.
- `serpentine`: uses a snake draft (red/blue/blue/red) so the strongest players are spread more evenly.
- `gosper`: searches for the split with the smallest rating difference using Gosper's hack for pools up to a
  configurable size, with a greedy swap fallback for larger pools.

The game server calls `GB_ScrambleTeams` from `gbans.inc`, passing the client indexes to include. The plugin filters
out bots, SourceTV, replay clients and players without a valid SteamID64, then sends the remaining players to the
backend. Spectators and unassigned players are included. The backend resolves the stats bucket and balancer itself,
looks up each player's internal rating (`mu`, using the calculator's starting value for unrated players), and returns
the teams. The gbans plugin then applies the team changes and invokes the caller's callback with the result.

Set `gb_debug_scramble` to `1` to log the resulting red/blu/waiting assignments (name, user ID and SteamID64) to the
server log. Failed scrambles are logged there as well.

## Stats refresh schedule

The aggregated stats views (daily, weekly, monthly, yearly and all-time summaries, per-map views and player ratings)
are refreshed by a background cron job. The schedule is configured from the same **Stats** settings tab as a standard 5-field cron
expression (minute, hour, day of month, month, day of week), evaluated in the server's local time. The default is
`5 0 * * *`, which refreshes the views daily at 00:05.

Changes take effect within about 30 seconds of saving; no restart is required. On a fresh installation the player
ratings are built once in the background at startup instead of waiting for the first scheduled run.
