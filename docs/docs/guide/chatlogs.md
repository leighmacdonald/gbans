# Chat Logs and Word Filters

## Chat logs

In-game chat is captured and made searchable. Enable the public view via
**Admin → Settings → General → Enable public chatlogs**.

- **Chat Logs** (`/chatlogs`) is a searchable view of captured chat messages with the
  server, timestamp, demo reference, SteamID, and the message.
- You can [report](reports.md) a player directly from a chat message.

## Word filters

Word flags and automatic actions for banned words in chat are managed under
**Admin → Filtered Words** and **Admin → Settings → Word Filters**.

Word filters are **disabled by default**. Enable the feature via
**Admin → Settings → Word Filters → Enable Word Filters** before any live chat logs or
usernames are checked.

- Each **filter** defines a pattern (plain text or regex) plus an **action** (kick, mute or
  ban), a **duration**, and a **weight**.
- **How it works** — chat logs and usernames received live from your servers (via the srcds
  log address) are checked against enabled filters in real time. Each match records a
  weighted warning for the player. Warnings are ephemeral and expire after the configured
  timeout. When the sum of a player's active warnings exceeds the configured maximum weight,
  the filter's action is applied automatically.
- **Flagged messages** — matches on live chat are recorded against the stored message once
  the match's demo is imported (run `gbans refresh filters` to backfill), which powers the
  "Flagged Only" view on the Chat Logs page.
- **Dry run mode** — run the filters without punishing, so you can tune your word list.
- **Discord notices** — optionally post filter events to a Discord channel.

## Live chat feed

The **Filtered Words** page (`/admin/filters`) also shows a live feed of in-game chat
messages as they arrive from your servers. Messages start streaming as soon as the page
is open and continue while the tab stays visible. You can pause the feed, and switch to a
single server or watch all of them at once. A short window of recent history is replayed
when you switch, since the feed is an in-memory buffer, older messages are not shown
(recent history remains available on the Chat Logs page after a demo is imported).
Team-only messages are tagged with a **Team** chip. The connection is kept alive while
chat is quiet and reconnects automatically (replaying any missed buffered messages) if it
is dropped by a proxy or network interruption.
