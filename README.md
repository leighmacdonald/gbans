# gbans

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Test, Build & Publish](https://github.com/leighmacdonald/gbans/actions/workflows/build.yml/badge.svg?branch=master)](https://github.com/leighmacdonald/gbans/actions/workflows/build.yml)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/f06234b0551a49cc8ac111d7b77827b2)](https://www.codacy.com/manual/leighmacdonald/gbans?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=leighmacdonald/gbans&amp;utm_campaign=Badge_Grade)
[![Go Report Card](https://goreportcard.com/badge/github.com/leighmacdonald/gbans)](https://goreportcard.com/report/github.com/leighmacdonald/gbans)
[![GoDoc](https://godoc.org/github.com/leighmacdonald/gbans?status.svg)](https://pkg.go.dev/github.com/leighmacdonald/gbans)
![Lines of Code](https://tokei.rs/b1/github/leighmacdonald/gbans)
[![Discord chat](https://img.shields.io/discord/704508824320475218)](https://discord.gg/YEWed3wY3F)

gbans is a centralized game administration system for Team Fortress 2 that incorporates a global ban system similar to
[sourcebans++](https://sbpp.dev) as well as some additional basic community tools like news, forums & wikis.
The project was initially only intended to be a more modern & secure replacement for sourcebans, but has evolved to
handle broader community aspects.

## Stability / Usage Notice

While we currently are [dogfooding](https://en.wikipedia.org/wiki/Eating_your_own_dog_food) the project on a
community with around 50 servers, I would not recommend non-developers use the project yet. It's still in fairly
major development mode and large sections are still incomplete or function but very rough.

Before we tag a 1.0.0 release, we will write some proper user-facing documentation. You can however look at the 
currently incomplete [docs](https://leighmacdonald.github.io/gbans/docs/INTRO).

## Features

- [ ] General
    - [x] Multi server support
    - [x] Global bans & mutes
    - [x] [Docker support](https://ghcr.io/leighmacdonald/gbans:latest)
- [ ] Import/Export of gbans databases
    - [ ] Backend linking of gbans services to enable use of other operators lists in real-time.
    - [x] [tf2_bot_detector](https://github.com/PazerOP/tf2_bot_detector) compatible player list exports.
- [x] Game support, support for other games would be possible, but is not currently planned.
    - [x] Team Fortress 2
- [ ] Blocking lists & types
    - [x] Valves source server banip config
    - [ ] Existing sourcebans database
    - [x] [CIDR/IP](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing) bans
    - [x] [tf2_bot_detector](https://github.com/PazerOP/tf2_bot_detector/blob/master/staging/cfg/playerlist.official.json)
    - [x] Arbitrary 3rd party CIDR block lists, Proxy/VPN/Datacenter bans
    - [ ] [FireHOL](https://github.com/firehol/blocklist-ipsets) databases
- [x] Database support
    - [x] Postgresql w/[PostGIS](https://postgis.net/) & [ip4r](https://github.com/RhodiumToad/ip4r) extensions
- [x] Centralized UDP based logging
    - [x] Message History
    - [x] Connection History
- [ ] Match Histories
    - [ ] Stats
        - [x] Overall
        - [x] Per Weapon
        - [ ] Per Maps
        - [ ] Per Server
        - [ ] Per Timeframe
        - [ ] Daily position changes
        - [ ] Charts
    - [ ] "Skill" based scrambles using historical match data
- [ ] SourceMod Plugin
    - [x] Game server authentication
    - [x] Restrict banned players from connecting
    - [x] Restrict muted/gagged players on join
    - [x] Dynamically load admins/mods/reserved slots from gbans
    - [x] Sourcemod admins-sql-* plugin compatibility.
    - [x] STV Recording with auto truncating methods to cleanup old demos
    - [ ] `/gb_ban <player_id|steam_id> duration Reason` Ban a user
    - [ ] `/gb_unban` Unban a previously banned user
    - [ ] `/gb_kick` Kick a user
    - [x] `/gb_mod or /mod <message>` Call for a mod on discord
- [x] Markdown formatting support for all multiline inputs with image uploading
- [x] Community Forum System
  - [x] Categories
  - [x] Forums
  - [x] Private / Restricted forums
  - [x] Signatures
  - [x] Active user display
  - [x] Advanced markdown editor interface
- [ ] User Interfaces
    - [ ] CLI
    - [x] Discord
    - [x] Web
        - Public / Authenticated
          - [x] Server Browser
          - [x] News
          - [x] Wiki
          - [x] User reports
          - [x] Ban Appeals
          - [x] SourceTV Downloads
          - [x] In-Game chat logs
          - [x] Server browser
        - Mods
          - [x] Edit Word Filters
          - [x] Ban player by steam/net/asn/group
          - [x] Current Reports
          - [x] Current Appeals
          - [x] Edit News
          - [x] Edit contests
          - [x] In-Game Vote logging
          - [x] Networking Queries
            - [x] Player IP History
            - [x] Players Sharing IPs
            - [x] Query IP information: Location/ASN/Proxy/Map
          - [x] External CIDR block lists editor
        - Admins
            - [x] Edit Servers
            - [x] Edit game moderator/admin permissions. Admins/Groups/Command overrides/Group immunities.

## Docker

Docker is recommended to run gbans. You can find the official docker images on
[ghcr.io](https://github.com/leighmacdonald/gbans/pkgs/container/gbans).

Assuming you have created your config file and have a database setup you can run it using something
like:

    docker run -it --rm -v `$(pwd)`/gbans.yml:/app/gbans.yml:ro ghcr.io/leighmacdonald/gbans:latest

There is also a docker-compose config you can use which provides the database as well.

    docker-compose -f docker/docker-compose.yml up --build --remove-orphans --abort-on-container-exit --exit-code-from gbans

## Documentation

For installation, configuration and usage instruction, developer notes, please see the [docs](https://leighmacdonald.github.io/gbans/docs/INTRO).

## Contributors

<!-- readme: contributors -start -->
<table>
	<tbody>
		<tr>
            <td align="center">
                <a href="https://github.com/leighmacdonald">
                    <img src="https://avatars.githubusercontent.com/u/591973?v=4" width="100;" alt="leighmacdonald"/>
                    <br />
                    <sub><b>Leigh MacDonald</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/Gcommer">
                    <img src="https://avatars.githubusercontent.com/u/243352?v=4" width="100;" alt="Gcommer"/>
                    <br />
                    <sub><b>Gcommer</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/rjdXpxigYxNT">
                    <img src="https://avatars.githubusercontent.com/u/168137561?v=4" width="100;" alt="rjdXpxigYxNT"/>
                    <br />
                    <sub><b>rjdXpxigYxNT</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/crescentrose">
                    <img src="https://avatars.githubusercontent.com/u/9057156?v=4" width="100;" alt="crescentrose"/>
                    <br />
                    <sub><b>Ivan</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/Mircoxi">
                    <img src="https://avatars.githubusercontent.com/u/95107897?v=4" width="100;" alt="Mircoxi"/>
                    <br />
                    <sub><b>Mircoxi</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/Tolfx">
                    <img src="https://avatars.githubusercontent.com/u/57797792?v=4" width="100;" alt="Tolfx"/>
                    <br />
                    <sub><b>Matthew</b></sub>
                </a>
            </td>
		</tr>
		<tr>
            <td align="center">
                <a href="https://github.com/Naleksuh">
                    <img src="https://avatars.githubusercontent.com/u/24828999?v=4" width="100;" alt="Naleksuh"/>
                    <br />
                    <sub><b>Naleksuh</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/camp3r101">
                    <img src="https://avatars.githubusercontent.com/u/10223311?v=4" width="100;" alt="camp3r101"/>
                    <br />
                    <sub><b>camp3r101</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/jack-avery">
                    <img src="https://avatars.githubusercontent.com/u/47289484?v=4" width="100;" alt="jack-avery"/>
                    <br />
                    <sub><b>Jack Avery</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/tdurieux">
                    <img src="https://avatars.githubusercontent.com/u/5577568?v=4" width="100;" alt="tdurieux"/>
                    <br />
                    <sub><b>Thomas Durieux</b></sub>
                </a>
            </td>
		</tr>
	<tbody>
</table>
<!-- readme: contributors -end -->