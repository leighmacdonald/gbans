# MGE Standings

MGE-mod is a SourceMod plugin that records 1v1 and 2v2 duel results. When it is enabled
(**Admin → Settings → General → Enable mge rankings**), gbans displays the duel standings
and match history from the plugin's `mgemod_*` tables.

- **Overall standings** (`/mge`) — rankings by rating with wins, losses and last played,
  plus links into the 1v1 and 2v2 history.
- **1v1 history** (`/mge/1v1`) — individual duels with winner, loser, both scores, win
  limit, game time, map and arena.
- **2v2 history** (`/mge/2v2`) — team duels with both players per side and the result.

MGE data is recorded by the plugin on your game server; gbans reads it and is read-only.
