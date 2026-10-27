# Servers

## Server browser

The public **Servers** page (`/servers`) lists your active game servers with their map,
player count, and — when [user location](../install/config.md#enabling-user-location) is
enabled — distance from the viewer. Players can sort and filter the list and copy a
connect address.

## Server management

**Admin → Servers** lets operators manage the registered game servers. Each server has:

- **Identity** — short name/tag, long name, address, port, and an optional internal
  address.
- **Authentication** — server auth key, RCON password, and log secret (used to ingest
  server logs).
- **Location** — region, country code, latitude and longitude (used for the distance
  column on the browser).
- **SDR** — an enable flag for [SDR](sdr.md) support.
- **Stats** — a per-server enable flag and the [stats bucket](stats.md) (game-mode) it
  plays in.
- **Seeding** — the Discord channel and role(s) used for [seed requests](discord.md#seed_roles)
  on that server.
- **Is Enabled** — whether the server is active.

**Admin → Server Logs** shows the raw console logs captured from your servers for
troubleshooting.
