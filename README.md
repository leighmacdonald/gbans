# gbans

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Test](https://github.com/leighmacdonald/gbans/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/leighmacdonald/gbans/actions/workflows/test.yml)
[![CodeQL](https://github.com/leighmacdonald/gbans/actions/workflows/codeql.yml/badge.svg?branch=master)](https://github.com/leighmacdonald/gbans/actions/workflows/codeql.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/leighmacdonald/gbans.svg)](https://pkg.go.dev/github.com/leighmacdonald/gbans)
[![Discord chat](https://img.shields.io/discord/704508824320475218)](https://discord.gg/YEWed3wY3F)

gbans is a centralized administration system for Team Fortress 2 communities. It
provides the tooling needed to run a set of game servers day-to-day: reports and appeals, chat logs,
player stats, SourceTV demos, forums, a wiki and Discord integration.

The project consists of a Go backend/API, a React web frontend (embedded into
the server binary), a SourceMod plugin and a PostgreSQL database using the
[PostGIS](https://postgis.net/) and [ip4r](https://github.com/RhodiumToad/ip4r)
extensions.

> **Status:** gbans is in active development and still pre-1.0. It is used in
> production, but some features are incomplete and internal APIs may change.

## Features

- **Bans & mutes** — Steam, IP/CIDR, ASN, group and whitelisting.
- **Moderation** — player reports, ban appeals, chat and connection logs, word
  filters and Discord log channels.
- **SourceMod plugin** — server authentication, ban enforcement, admin, group and
  override synchronisation, in-game menus and player clientprefs.
- **Stats** — player and weapon statistics generated from match demos with
  [tf2_demostats](https://github.com/leighmacdonald/tf2_demostats).
- **SourceTV** — automatic demo recording, storage and download from the web UI.
- **Community** — news, forums and a wiki, all managed from the web UI.
- **Web UI** — server browser, ban management and admin screens for SourceMod
  admins, groups and permissions.

More detail on each of these is available in the [documentation](#documentation).

## Quick start

Everything is compiled into a single binary; the only external dependency is
PostgreSQL with the PostGIS and ip4r extensions. Docker is the recommended way
to run gbans:

```shell
cp gbans_example.yml gbans.yml   # set the database DSN, owner, steam key, etc.
docker run -d --restart unless-stopped \
  -p 6006:6006 \
  -v "$(pwd)/gbans.yml:/app/gbans.yml:ro" \
  --name gbans \
  ghcr.io/leighmacdonald/gbans:latest
```

See the [installation guide](https://leighmacdonald.github.io/gbans/docs/install/install)
for requirements, database setup and reverse proxy configuration, and the
[configuration reference](https://leighmacdonald.github.io/gbans/docs/install/config)
for a description of every setting.

## Documentation

- **Operators** — [installation](https://leighmacdonald.github.io/gbans/docs/install/install),
  [configuration](https://leighmacdonald.github.io/gbans/docs/install/config),
  [SourceMod plugin](https://leighmacdonald.github.io/gbans/docs/guide/sourcemod),
  [bans](https://leighmacdonald.github.io/gbans/docs/guide/bans),
  [moderation](https://leighmacdonald.github.io/gbans/docs/guide/moderation),
  [Discord](https://leighmacdonald.github.io/gbans/docs/guide/discord),
  [stats](https://leighmacdonald.github.io/gbans/docs/guide/stats),
  [SourceTV](https://leighmacdonald.github.io/gbans/docs/guide/sourcetv) and
  [SDR](https://leighmacdonald.github.io/gbans/docs/guide/sdr).
- **Developers** — [development environment](https://leighmacdonald.github.io/gbans/docs/devel/devel),
  [building](https://leighmacdonald.github.io/gbans/docs/devel/build) and
  [updating the documentation](https://leighmacdonald.github.io/gbans/docs/devel/docs).

## Development

The backend, frontend and SourceMod plugin can be built with
[just](https://github.com/casey/just):

```shell
just db        # start postgres in docker
just dev       # database + backend + frontend watchers (zellij)
just run       # run the backend only
just check     # linters, static analysis and type checks
just test      # backend and frontend test suites
just generate  # regenerate code from the protobuf definitions
```

Requirements are Go 1.27+, Node.js 18.17+ with pnpm, and SourceMod 1.12+ to
compile the plugin. See the
[development guide](https://leighmacdonald.github.io/gbans/docs/devel/devel)
for the full setup.

## License

gbans is released under the [MIT license](LICENSE).
