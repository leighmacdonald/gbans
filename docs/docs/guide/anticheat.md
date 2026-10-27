# Anticheat

gbans records and automatically acts on in-game anticheat (STAC) detections from your
servers.

## How it works

Your servers emit STAC anticheat events — either over the UDP log listener or pulled from
STAC log files over SSH (configured under **Admin → Settings → Anticheat** and **SSH**).
Each event is stored with the player, the detection type, a demo reference, and the raw
log line.

## Automatic action

Under **Admin → Settings → Anticheat** you configure:

- **Enabled** and the **punishment strategy** — `gag`, `kick`, or `ban` — plus its
  **duration**.
- Per-detection thresholds (aim snaps, silent aim, bhop, fake angles, cmd-number spikes,
  too many connections, cheat CVars, out-of-bounds CVars, invalid user cmds). When a
  player exceeds a threshold for a detection type, the configured punishment is applied.

Detections that exceed their threshold are also posted to your configured Discord
anticheat channel.

## Anticheat logs

**Admin → Anticheat Logs** shows the full history of detections: player, server, detection
type, summary, the associated demo (for review), and how many times that detection has
triggered for the player. Filter by player, server, detection type, or search the summary.
