# gbans

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
- [x] (Docker support)[https://hub.docker.com/repository/docker/leighmacdonald/gbans]
- [ ] ACME ([Lets encrypt](https://letsencrypt.org/) / [Zero SSL](https://zerossl.com/)) protocol support for automatic SSL certificates
- [ ] SourceMod Plugin
    - [x] Game server authentication
    - [ ] `gb_ban <player_id|steam_id> duration Reason` Ban a user
    - [ ] `gb_unban` Unban a previously banned user
    - [ ] `gb_kick` Kick a user
    - [ ] `mod` Call for a mod 
- [ ] User Interfaces
    - [x] Discord
    - [ ] Web
- [ ] Game server logs
   - [x] Remote relay client `gbans relay -h`
   - [x] Parsing  
   - [x] Indexing 
   - [ ] Querying
    
## Docker

Docker is recommended to run gbans. You can find the official docker images at 
(dockerhub)[https://hub.docker.com/repository/docker/leighmacdonald/gbans].

Assuming you have created your config file and have a database setup you can run it using something
like:

    docker run -it --rm -v `$(pwd)`/gbans.yml:/app/gbans.yml:ro leighmacdonald/gbans:latest

There is also a docker-compose config you can use which provides the database as well.

    docker-compose -f docker/docker-compose.yml up --build --remove-orphans --abort-on-container-exit --exit-code-from gbans

## Documentation

For installation, configuration and usage instruction, please see the [docs](docs) directory.
