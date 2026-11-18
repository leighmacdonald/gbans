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

Changing the algorithm **requires a restart** because the rating service is constructed at startup. The next stats
refresh rebuilds every rating with the new algorithm, so switching rebases all players; existing rating history is
replaced as part of the same replay. Display values are calculated by the selected algorithm and persisted with each
rating, so the leaderboard view always uses the correct scale for the algorithm that produced it.

Ratings are rebuilt from scratch on every stats refresh by replaying every rated match in chronological order, so
late-imported demos and deleted matches are handled automatically. The current rating is exposed by
`StatsService.PlayerRating`, rating-over-time snapshots by `StatsService.PlayerRatingHistory`, and the per-bucket
ranking by `StatsService.RatingLeaderboard`. All three require the `stats_read` permission.

## Stats refresh schedule

The aggregated stats views (daily, weekly, monthly, yearly and all-time summaries, per-map views and player ratings)
are refreshed by a background cron job. The schedule is configured from the same **Stats** settings tab as a standard 5-field cron
expression (minute, hour, day of month, month, day of week), evaluated in the server's local time. The default is
`5 0 * * *`, which refreshes the views daily at 00:05.

Changes take effect within about 30 seconds of saving; no restart is required. On a fresh installation the player
ratings are built once in the background at startup instead of waiting for the first scheduled run.
