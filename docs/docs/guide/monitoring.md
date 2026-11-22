---
sidebar_position: 15
---

# Monitoring

gbans exposes a Prometheus `/metrics` endpoint on a dedicated internal HTTP listener and ships a
ready-to-import Grafana dashboard.

## Enabling the exporter

Add the following to `gbans.yml` and restart gbans (see [Base Configuration](../install/config.md#metrics--profiling)):

```yaml
prometheus_enabled: true
http_internal_host: 127.0.0.1
http_internal_port: 6605
```

The endpoint is then available at `http://<http_internal_host>:<http_internal_port>/metrics`. It is
deliberately separate from the public web/RPC port. If Prometheus runs on another host, bind
`http_internal_host` to an internal interface that Prometheus can reach — avoid `0.0.0.0` on a
publicly exposed machine, since pprof is served on the same listener when enabled.

Example Prometheus scrape config:

```yaml
scrape_configs:
  - job_name: gbans
    scrape_interval: 30s
    static_configs:
      - targets: ["gbans-host:6605"]
```

## Importing the Grafana dashboard

1. In Grafana, open **Dashboards → New → Import**.
2. Upload [`monitoring/grafana/gbans-dashboard.json`](https://github.com/leighmacdonald/gbans/blob/master/monitoring/grafana/gbans-dashboard.json)
   from the repository, or paste its contents.
3. Select your Prometheus datasource when prompted (the dashboard uses a `DS_PROMETHEUS` variable),
   then click **Import**.

The dashboard has a **Server** variable populated from `gbans_server_players`, so you can filter the
per-server panels to one or more game servers.

## What the dashboard shows

| Section | Panels |
| --- | --- |
| Overview | Build version, enabled servers, human players, HTTP request rate, log-event rate, uptime |
| Game Servers | Players per server, busiest servers, connects/disconnects, RCON commands |
| HTTP Traffic | Requests by route, 5xx rate, p50/p95/p99 latency, top routes |
| Game Log Events | Chat, kills/damage/healing by weapon, shots fired vs hit, maps played, class distribution |
| Go Runtime | Goroutines, memory, CPU |

The **Game Log Events** panels only contain data when the UDP game-log listener is receiving logs;
they are empty on installations without log ingestion. The HTTP request panels are labelled by route
with dynamic ID segments collapsed (for example `/ban/{id}`), keeping label cardinality bounded.

## Available metrics

Application metrics (all prefixed `gbans_`):

| Metric | Type | Labels | Description |
| --- | --- | --- | --- |
| `gbans_build_info` | gauge | `version` | Build version (always 1) |
| `gbans_servers_total` | gauge | — | Enabled game servers |
| `gbans_players_total` | gauge | — | Human players across servers |
| `gbans_server_players` | gauge | `server_name` | Human players on each server |
| `gbans_http_requests_total` | counter | `method`, `route`, `code` | Handled HTTP requests |
| `gbans_http_request_duration_seconds` | histogram | `method`, `route` | HTTP request latency |
| `gbans_game_log_events_total` | counter | — | Log lines consumed |
| `gbans_game_chat_total` | counter | `team_say` | Chat messages |
| `gbans_game_kills_total` | counter | `weapon` | Kills from log events |
| `gbans_game_damage_total` | counter | `weapon` | Damage from log events |
| `gbans_game_healing_total` | counter | `weapon` | Healing from log events |
| `gbans_game_shot_fired_total` | counter | `weapon` | Shots fired |
| `gbans_game_shot_hit_total` | counter | `weapon` | Shots hit |
| `gbans_player_class_total` | counter | `class` | Spawns by class |
| `gbans_player_connected_total` | counter | `server_name` | Player connects |
| `gbans_player_disconnected_total` | counter | `server_name` | Player disconnects |
| `gbans_rcon_total` | counter | `server_name` | RCON commands |
| `gbans_map_played_total` | counter | `map` | Map plays |
| `gbans_player_count` | histogram | `server_name` | Player count samples |

Standard Go and process collectors are registered as well (`go_goroutines`, `go_memstats_*`,
`go_gc_duration_seconds`, `process_cpu_seconds_total`, `process_resident_memory_bytes`,
`process_start_time_seconds`).
