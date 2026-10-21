# Stats Overview

Stats are generated for many game events. They are summarized into atomic matches and stored in the database. The
design goal is to have a cross between [logs.tf](https://logs.tf)
and [hlstatsx:ce](https://github.com/A1mDev/hlstatsx-community-edition).

## Game Stats page

The `/stats` page (requires the `stats_read` permission) shows leaderboards per stats bucket for a chosen time
window (day, week, month, year, or all time), overall or broken down by kills, healing, weapon, or class.

When the signed-in player has stats in the selected window, their rank is shown in the page header. Clicking it
jumps the leaderboard table to the page containing their row and briefly highlights it.

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
