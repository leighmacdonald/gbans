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
- **Colored output** — each message category has its own color, making it easy to spot chat, kills, world events
  and server metadata at a glance.

The page requires the `PERMISSION_SERVER_READ` permission.
