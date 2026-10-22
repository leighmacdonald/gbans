# Stats Overview

Stats are generated for many game events. They are summarized into atomic matches and stored in the database. The
design goal is to have a cross between [logs.tf](https://logs.tf)
and [hlstatsx:ce](https://github.com/A1mDev/hlstatsx-community-edition).

## Game Stats page

The `/stats` page (requires the `stats_read` permission) shows leaderboards per stats bucket for a chosen time
window (day, week, month, year, or all time), overall or broken down by kills, healing, weapon, or class.

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
