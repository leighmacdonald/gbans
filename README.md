# gbans

gbans is intended to be a more modern & secure replacement 
for [sourcebans](https://github.com/GameConnect/sourcebansv1) / [sourcebans++](https://sbpp.dev).

## Primary differences from sourcebans++

- No direct SQL queries across networks. Exposing MySQL to the internet is a very poor security practice. You can 
of course mitigate this with firewalls and sql accounts with ip restrictions or VPNs, but the majority of 
server admins will not ever do this.
- Game servers authenticate with the gban server upon startup of the plugin. Subsequent requests will use the returned
authentication token.
- Communication over HTTPS
- Discord bot integration for administration & announcements.
- Built using [Go](https://golang.org/) & [SQLite](https://www.sqlite.org/index.html). This enables trivial deployment as its just a matter of running the binary. It has a built-in 
webserver that is safe to directly expose to the internet. This means its not necessary to setup MySQL, 
Nginx/Apache and PHP on your server.
- Non-legacy codebase that is (hopefully) not a nightmare to hack on. Sourcebans++, while updated, is still very clearly a legacy PHP codebase. It uses no framework or real conventions, Still uses globals, Mixes PHP/JS code, Uses tables for layout. I am in no way trying to put them down, they have done a good job volunteering their time over the years, But i think its time to move on from this legacy stuff.

## Features

- [ ] Import of existing sourcebans database
- [ ] Import/Export of gban databases
- [ ] 3rd party ban lists (eg: [tf2_bot_detector](https://github.com/PazerOP/tf2_bot_detector/blob/master/staging/cfg/playerlist.official.json))
- [x] Discord integration
- [x] Multi server support
- [x] Global bans
- [x] Subnet & IP bans (CIDR)
- [x] Database support
    [x] Postgresql
    [?] MySQL, if there is demand (or pr) for it
    [?] Sqlite, if there is demand (or pr) for it
 - [x] Docker support
    - [ ] Published official images
- [ ] ACME ([Lets encrypt](https://letsencrypt.org/) / [Zero SSL](https://zerossl.com/)) protocol support for automatic SSL certificates
- [x] SourceMod Plugin
    - [x] Game server authentication
    - [ ] `g_ban <player_id|steam_id> duration Reason` Ban a user
    - [ ] `g_unban` Unban a previously banned user
    - [ ] `g_kick` Kick a user
    - [ ] `/mod` Call for a mod 

## Documentation

For installation, configuration and usage instruction, please see the [docs](docs) directory.
