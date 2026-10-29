# Server Logs

The **Server Logs** admin page (`/admin/serverlogs`) shows a live, terminal style stream of each server's log
output. Log lines are received over the UDP log listener (the same `sv_logsecret` based feed used for everything
else), parsed, classified by message type, and colored accordingly.

:::info

The view is backed by an in-memory buffer that keeps the **last 10,000 lines per server**. History does not
survive a gbans restart, and lines from before the buffer filled up are not available. A longer term copy of the
log lines is also kept in the database (purged after 24 hours).

:::

## Features

- **Server selector** — pick which server instance to tail.
- **Message type filters** — lines are classified into categories (chat, kills, damage, players, world, server,
  votes, other). Toggle categories on or off to narrow the stream, and use the text filter to search within the
  buffered lines.
- **Pause** — pauses the live updates. Lines that arrive while paused are discarded and a counter shows how many
  were skipped.
- **Auto-scroll** — keeps the view pinned to the newest line. It disables itself automatically when you scroll up
  to review older output, and can be re-enabled with the button.
- **Clear logs** — users with the `PERMISSION_SERVER_WRITE` permission see a button that empties the selected
  server's in-memory buffer. The live stream keeps flowing afterwards; only the already-buffered history is
  removed.
- **Colored output** — each message category has its own color, and player names, quoted values, timestamps and
  numbers are highlighted within lines, making it easy to spot chat, kills, world events and server metadata at a
  glance.
- **Player quick links** — player references such as `"Hacksaw<12><[U:1:68745073]><Red>"` are clickable. Clicking
  one opens a menu to filter the stream to that player, open their local profile or Steam profile, copy their
  SteamID 64, or jump straight to their chat logs, SourceTV history, connection history, ban history and report
  history (permission dependent).
- **RCON console** — users with the `PERMISSION_SERVER_WRITE` permission see an input box below the terminal to
  send an RCON command (e.g. `sm_listme`, `changelevel cp_badlands`) to the selected server. The command output,
  if any, is shown under the input.

The page requires the `PERMISSION_SERVER_READ` permission.
