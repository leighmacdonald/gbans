# Discord Reference

Here you will find an outline of the various available discord bot commands you can use.

This of course assumes you have discord integration enabled. See [INSTALL.md](INSTALL.md) for details on
how to enable it.

## Overview

The bot currently has a few caveats for proper usage. Please make sure you are aware of these. They can and will
be changed to be more flexible in the future.

- It uses discords new [slash commands](https://discord.com/developers/docs/interactions/slash-commands) to perform
  the commands, no ! commands.
- You have an admin/mod role that the bot will operate under. This means it is your responsibility to ensure the people with access to this
channel are trustworthy. Only users under this role will be allowed to use privileged commands.
- There is currently no direct RCON command for the bot. gbans will use RCON internally, but it's not exposed to the bot
yet. Once finer access controls get integrated it will possibly, also be added.
- All bans, steamid & network, apply to all servers, there is currently no way to ban on specific servers only.

### Common Arg/Term Reference

These are the more thorough details of the arguments used in the bot commands below.
 
`user_identifier` This can refer to either an in game name, steam id of any variety, or a profile url. If using a name,
it will match the first name found with a partial match. If a command takes a specific
server-id, it will search all servers concurrently for a match and stop on the first match if multiple 
are found. Name will only work when the user is actually currently in a server, you must use a steamid to do a offline ban.

`duration` Duration accepts either a duration string, or `0` for permanent. It is based on the formatting in the
golang [time.Duration](https://golang.org/pkg/time/#ParseDuration) function, its however expanded to support more
fractional units larger than hours. 

Current supported formats, N being the count, eg: `10s` for 10 seconds:

- `Ns` seconds
- `Nm` minutes
- `Nh` hours
- `Nd` days
- `Nw` weeks
- `NM` months
- `Ny` years

## Command Reference

### /ban <user_identifier> <duration> \[reason\]

Ban a user from the server 

### /banip <cidr> <duration> \[reason\]

Ban a network block or IP. To ban a single ip use a /32 block, eg: `10.11.12.100/32`. To ban a 
subnet block use a larger mask. eg: `10.11.12.0/24`. This would block all ips between: `10.11.12.1 - 10.11.12.255`

Banning a network:
```
 user: /banip 1.2.3.0/24 5m Example network ban
gbans: IP ban created successfully
```

Banning an IP:
```
 user: /banip 1.2.3.4/32 5m Example IP ban
gbans: IP ban created successfully
```

### /find <user_identifier>

Locate a player by name or steamid, returning their steamid and current server-id.

```
 user: /find tim
gbans: Found player Recliner_Tim (76561198282517317) @ us-1
```

### /mute <user_identifier> <duration> \[reason\]


### /check <steamid>

Check the current ban state of a steamid

```
 user: /check 76561199093644873

```

### /unban <steamid>

Unbans a previously banned user ahead of the expiration time.

```
 user: /unban 76561199093644873
gbans: User ban is now inactive
```

### /kick <user_identifier> \[reason\]

Kick a currently playing user from the server

### /players <server-id>

Returns a table of player info similar to a `status` command.

Columns: `<player-id> <steam-id> <ip> <name>`

```/players example-1

┌────────────────┬───────────────────┬───────────────────────────────┐
│ IP             │ STEAM64           │ NAME                          │
├────────────────┼───────────────────┼───────────────────────────────┤
│ 10.10.10.10    │ 76561197967968980 │ ZEKKER                        │
│ 10.10.10.10    │ 76561197999666480 │ sorbent                       │
│ 10.10.10.10    │ 76561198026039048 │ ShiningInTheDarkness          │
│ 10.10.10.10    │ 76561198035350625 │ sambtaylor                    │
│ 10.10.10.10    │ 76561198045964573 │ it's ok to be sad, or >>>mad! │
│ 10.10.10.10    │ 76561198068000747 │ DJ Absolute Garbage           │
│ 10.10.10.10    │ 76561198071771352 │ Sub Zer0                      │
│ 10.10.10.10    │ 76561198078041126 │ Jayty07                       │
│ 10.10.10.10    │ 76561198083242298 │ orthotic horse shoes          │
│ 10.10.10.10    │ 76561198087730829 │ smart fella                   │
│ 10.10.10.10    │ 76561198099295077 │ Stapler                       │
│ 10.10.10.10    │ 76561198102116420 │ rat                           │
│ 10.10.10.10    │ 76561198116511493 │ Kris                          │
│ 10.10.10.10    │ 76561198125608211 │ tudrle                        │
│ 10.10.10.10    │ 76561198127462723 │ BasherRay                     │
│ 10.10.10.10    │ 76561198135535301 │ Pootis                        │
│ 10.10.10.10    │ 76561198140898046 │ Sowsar                        │
│ 10.10.10.10    │ 76561198162541373 │ SmokedApplee                  │
│ 10.10.10.10    │ 76561198165530739 │ NotAWeeb                      │
│ 10.10.10.10    │ 76561198184393819 │ Danky Licker                  │
│ 10.10.10.10    │ 76561198271822342 │ /Æ\ tim                       │
│ 10.10.10.10    │ 76561198281773224 │ CCAT                          │
│ 10.10.10.10    │ 76561198286757016 │ chuck marten                  │
│ 10.10.10.10    │ 76561199045827269 │ connor                        │
└────────────────┴───────────────────┴───────────────────────────────┘
```

### /psay <server-id> <name/steamid> <message>

Sends a private chat message to a single target. This just calls the [sourcemod command](https://wiki.alliedmods.net/Admin_Commands_(SourceMod)) 
`sm_psay`

### /csay <server-id | *> <message>

Sends a centered message to all players. This just calls the [sourcemod command](https://wiki.alliedmods.net/Admin_Commands_(SourceMod)) 
`sm_csay`

### /say <server-id | *> <message>

Sends a say-chat message to all players.  This just calls the [sourcemod command](https://wiki.alliedmods.net/Admin_Commands_(SourceMod)) 
`sm_say`

### /servers

Returns a table of current server details with these columns: `<server-id> <server-name> <map> <players>`

```
    /servers
    
    ┌──────┬────────────────────────────┬──────────────────────┬─────────┐
    │ ID   │ NAME                       │ CURRENT MAP          │ PLAYERS │
    ├──────┼────────────────────────────┼──────────────────────┼─────────┤
    │ eu-2 │ Uncletopia | Berlin        │ koth_lazarus         │ 0/24    │
    │ us-2 │ Uncletopia | Chicago       │ pl_upward            │ 24/24   │
    │ us-6 │ Uncletopia | Dallas        │ pl_badwater          │ 23/24   │
    │ eu-1 │ Uncletopia | Frankfurt     │ pl_badwater          │ 21/24   │
    │ us-1 │ Uncletopia | Los Angeles   │ pl_upward            │ 23/24   │
    │ us-4 │ Uncletopia | New York City │ cp_snakewater_final1 │ 24/24   │
    │ us-3 │ Uncletopia | San Francisco │ koth_sawmill         │ 20/24   │
    │ us-5 │ Uncletopia | Seattle       │ pl_upward            │ 22/24   │
    │ au-1 │ Uncletopia | Sydney        │ koth_suijin          │ 0/24    │
    └──────┴────────────────────────────┴──────────────────────┴─────────┘
```
