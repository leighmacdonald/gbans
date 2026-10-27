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

- Each **filter** defines a pattern (plain text or regex) plus an **action**, a
  **duration**, and a **weight**.
- **How it works** — each time a player's message matches a filter, a weighted warning is
  recorded for them. Warnings are ephemeral and expire after the configured timeout. When
  the sum of a player's active warnings exceeds the configured maximum weight, the
  configured action is applied automatically.
- **Dry run mode** — run the filters without punishing, so you can tune your word list.
- **Discord notices** — optionally post filter events to a Discord channel.
