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

## Stats refresh schedule

The aggregated stats views (daily, weekly, monthly, yearly and all-time summaries, plus per-map views) are refreshed
by a background cron job. The schedule is configured from the same **Stats** settings tab as a standard 5-field cron
expression (minute, hour, day of month, month, day of week), evaluated in the server's local time. The default is
`5 0 * * *`, which refreshes the views daily at 00:05.

Changes take effect within about 30 seconds of saving; no restart is required.

## Compared with hlstatsx:ce

- Considerably better scaling performance:
  - Matches are committed to the database in a single transaction instead of immediately upon incoming events
  - Demo parser is written in rust and generally quite fast

:::note

TODO

:::

:::info

At some point in the future we plan to introduce a special "stats only" mode of gbans which is designed to
effectively act as a replacement for hlstatx, without all the extra community aspects enabled. Even with our incomplete
stats system, we had many operators asking if this would be a potential option.

:::
