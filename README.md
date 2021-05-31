# gbans

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Test, Build & Publish](https://github.com/leighmacdonald/gbans/actions/workflows/build.yml/badge.svg?branch=master)](https://github.com/leighmacdonald/gbans/actions/workflows/build.yml)
[![release](https://github.com/leighmacdonald/gbans/actions/workflows/release.yml/badge.svg?event=release)](https://github.com/leighmacdonald/gbans/actions/workflows/release.yml)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/f06234b0551a49cc8ac111d7b77827b2)](https://www.codacy.com/manual/leighmacdonald/gbans?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=leighmacdonald/gbans&amp;utm_campaign=Badge_Grade)
[![Maintainability](https://api.codeclimate.com/v1/badges/4e3242de961462b0edc7/maintainability)](https://codeclimate.com/github/leighmacdonald/gbans/maintainability)
[![Go Report Card](https://goreportcard.com/badge/github.com/leighmacdonald/gbans)](https://goreportcard.com/report/github.com/leighmacdonald/gbans)
[![GoDoc](https://godoc.org/github.com/leighmacdonald/gbans?status.svg)](https://pkg.go.dev/github.com/leighmacdonald/gbans)
![Lines of Code](https://tokei.rs/b1/github/leighmacdonald/gbans)
[![Discord chat](https://img.shields.io/discord/704508824320475218)](https://discord.gg/YEWed3wY3F)

gbans is intended to be a more modern & secure replacement 
for [sourcebans](https://github.com/GameConnect/sourcebansv1) / [sourcebans++](https://sbpp.dev).

## Primary differences from sourcebans++

- No direct SQL queries across networks. Exposing MySQL to the internet is a very poor security practice. You can 
of course mitigate this with firewalls and sql accounts with ip restrictions or VPNs, but the majority of 
server admins will not ever do this.
- Game servers authenticate with the gbans server upon startup of the plugin. Subsequent requests will use the returned
authentication token.
- Communication over HTTPS
- Discord bot integration for administration & announcements.
- Built using [Go](https://golang.org/) & [PostgreSQL](https://www.postgresql.org/). It has a built-in 
webserver that is safe to directly expose to the internet. This means its not necessary to setup MySQL, 
Nginx/Apache and PHP on your server.
- Non-legacy codebase that is (hopefully) not a nightmare to hack on.

## Features

- [ ] Import of existing sourcebans database
- [ ] Import/Export of gbans databases
- [ ] Backend linking of gbans services to enable use of other operators lists in real-time.
- [x] Game support
   - [x] Team Fortress 2
- [ ] 3rd party ban lists 
   - [x] [tf2_bot_detector](https://github.com/PazerOP/tf2_bot_detector/blob/master/staging/cfg/playerlist.official.json)
   - [ ] Known VPN Networks
   - [ ] Known non-residential addresses 
   - [ ] Known proxies
- [x] Multi server support
- [x] Global bans
- [x] Subnet & IP bans (CIDR)
- [x] Database support
  - [x] Postgresql w/PostGIS
- [x] [Docker support](https://hub.docker.com/repository/docker/leighmacdonald/gbans)
- [ ] ACME ([Lets encrypt](https://letsencrypt.org/) / [Zero SSL](https://zerossl.com/)) protocol support for automatic SSL certificates
- [ ] SourceMod Plugin
    - [x] Game server authentication
    - [ ] `/gb_ban <player_id|steam_id> duration Reason` Ban a user
    - [ ] `/gb_unban` Unban a previously banned user
    - [ ] `/gb_kick` Kick a user
    - [x] `/gb_mod or /mod <message>` Call for a mod on discord
- [ ] User Interfaces
    - [x] Discord
    - [ ] Web
- [ ] Game server logs
   - [x] Remote relay agent `gbans relay -h`
   - [x] Parsing  
   - [x] Indexing 
   - [ ] Querying
    
## Docker

Docker is recommended to run gbans. You can find the official docker images at 
[dockerhub](https://hub.docker.com/repository/docker/leighmacdonald/gbans).

Assuming you have created your config file and have a database setup you can run it using something
like:

    docker run -it --rm -v `$(pwd)`/gbans.yml:/app/gbans.yml:ro leighmacdonald/gbans:latest

There is also a docker-compose config you can use which provides the database as well.

    docker-compose -f docker/docker-compose.yml up --build --remove-orphans --abort-on-container-exit --exit-code-from gbans

## Documentation

For installation, configuration and usage instruction, please see the [docs](docs) directory.
