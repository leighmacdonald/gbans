# Discord Reference

Here you will find an outline of the various available discord bot commands you can use.

This of course assumes you have discord integration enabled. See [INSTALL.md](INSTALL.md) for details on
how to enable it.

## Overview

The bot currently have a few caveats for proper usage. Please make sure you are aware of these. They can and will
be changed to be more flexible in the future.

- You have an admin/mod channel that the bot will operate under. The bot currently does *NOT* check for any
special discord priviledges/roles. This means it is your responsibility to ensure the people with access to this
channel are trustworthy.
- It can join many servers, but it will only respond in channels with matching configured `Channel ID`s. Right click
the channel you want and select `Copy ID` to get your channel id.
- There is currently no direct RCON command for the bot. gbans will use RCON internally, but its not exposed to the bot
yet. Once finer access controls get integrated it will also be added.
- You can change the prefix operator `!` in the config.
- All bans, steamid & network, apply to all servers, there is currently no way to ban on specific servers only.

### Common Arg/Term Reference

These are the more thorough details of the arguments used in the bot commands below.
 
`<name/steamid>` This can refer to either an in game name, or a steam id. If using a name
it will match the first name found with a partial match. Unless the command takes a specific
server-id, it will search all servers concurrently for a match and stop on the first match if multiple 
are found. Name will only work when the user is actually currently in a server, you must use a steamid to do a offline ban.
The steamid can be any steamid format ([reference](https://pkg.go.dev/github.com/leighmacdonald/steamid@v1.2.0/steamid)), 
it will be formatted as needed in different places internally. Most commands will output steam64 based ids, there is not currently
a way to change this.

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

- `<argument>` These are required arguments
- `[argument]` These are optional arguments

### !help \[command\]

If no command it specified, it will return a list of available commands. If a command it provided, it
will provide basic description and usage info.

```
 user: !help
gbans: Available commands (!help <command>): !ban, !banip, !check, !csay, !find, !help, !kick, !mute, !players, !psay, !say, !servers, !unban
```

```
 user: !help ban
gbans: Ban a player -- !ban <name/id> <duration> [reason]
```

### !ban <name/steamid> <duration> \[reason\]

Ban a user from the server 

### !banip <cidr> <duration> \[reason\]

Ban a network block or IP. To ban a single ip use a /32 block, eg: `10.11.12.100/32`. To ban a 
subnet block use a larger mask. eg: `10.11.12.0/24`. This would block all ips between: `10.11.12.1 - 10.11.12.255`

Banning a network:
```
 user: !banip 1.2.3.0/24 5m Example network ban
gbans: IP ban created successfully
```

Banning an IP:
```
 user: !banip 1.2.3.4/32 5m Example IP ban
gbans: IP ban created successfully
```

### !find <name/steamid>

Locate a player by name or steamid, returning their steamid and current server-id.

```
 user: !find tim
gbans: Found player Recliner_Tim (76561198282517317) @ us-1
```

### !mute <name/steamid> <duration> \[reason\]


### !check <steamid>

Check the current ban state of a steamid

```
 user: !check 76561199093644873
gbans: [76561199093644873] Banned: true -- Muted: false -- IP: N/A -- Expires In: 9m49 Reason: Example reason
```

### !unban <steamid>

Unbans a previously banned user ahead of the expiration time.

```
 user: !unban 76561199093644873
gbans: User ban is now inactive
```

### !kick <name/steamid> \[reason\]

Kick a currently playing user from the server

### !players <server-id>

Returns a table of player info similar to a `status` command.

Columns: `<player-id> <steam-id> <ip> <name>`

```
 user: !players example-1
gbans: 
    323 76561198153543927 100.100.100.100 [Monsieur Marley]
    325 76561198067893730 100.100.100.100 [benny harvey]
    382 76561198043799102 100.100.100.100 [HaskellCurry]
    377 76561198066254434 100.100.100.100 [♥ Annie ♥]
    366 76561198271822342 100.100.100.100 [Recliner_Tim]
    373 76561198049372532 100.100.100.100 [Vesper]
...
```

### !psay <server-id> <name/steamid> <message>

Sends a private chat message to a single target. This just calls the [sourcemod command](https://wiki.alliedmods.net/Admin_Commands_(SourceMod)) 
`sm_psay`

### !csay <server-id> <message>

Sends a centered message to all players. This just calls the [sourcemod command](https://wiki.alliedmods.net/Admin_Commands_(SourceMod)) 
`sm_csay`

### !say <server-id> <message>

Sends a say-chat message to all players.  This just calls the [sourcemod command](https://wiki.alliedmods.net/Admin_Commands_(SourceMod)) 
`sm_say`

### !servers

Returns a table of current server details with these columns: `<server-id> <server-name> <map> <players>`

```
 user: !servers
gbans:
    us-3 -- Uncletopia | San Francisco -- pl_borneo           -- 24/32
    us-1 -- Uncletopia | Los Angeles   -- pl_frontier_final   -- 24/32
    us-2 -- Uncletopia | Chicago       -- pl_badwater_pro_v12 -- 24/32
    us-4 -- Uncletopia | New York City -- pl_barnblitz        -- 24/32
    eu-2 -- Uncletopia | Berlin        -- pl_badwater_pro_v12 -- 17/32
    eu-1 -- Uncletopia | Frankfurt     -- pl_badwater_pro_v12 -- 20/32
```
