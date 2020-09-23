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
- Built-in discord integration for ban/kick announcements and command to call an admin from in game.
- Built using [Go](https://golang.org/) & [SQLite](https://www.sqlite.org/index.html). This enables trivial deployment as its just a matter of running the binary. It has a built-in 
webserver that is safe to directly expose to the internet. This means its not necessary to setup MySQL, 
Nginx/Apache and PHP on your server.

## Features

- [ ] Import of existing sourcebans database
- [ ] Import/Export of gban databases
- [ ] 3rd party ban lists (eg: [tf2_bot_detector](https://github.com/PazerOP/tf2_bot_detector/blob/master/staging/cfg/playerlist.official.json))
- [ ] Discord notifications
- [ ] Multi server support
- [ ] Global and local server bans
- [ ] Official docker images
- [ ] ACME ([Lets encrypt](https://letsencrypt.org/) / [Zero SSL](https://zerossl.com/)) protocol support for automatic SSL certificates
- [ ] SourceMod Plugin
    - [x] Game server authentication
    - [ ] `g_ban <player_id|steam_id> duration Reason` Ban a user
    - [ ] `g_unban` Unban a previously banned user
    - [ ] `g_kick` Kick a user
    - [ ] `/mod` Call for a mod 
