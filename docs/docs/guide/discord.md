# Discord Reference

Here you will find an outline of the various available discord bot commands you can use.

This of course assumes you have discord integration enabled. See [INSTALL.md](../install/install) for details on
how to enable it.

## Overview

The bot currently has a few caveats for proper usage. Please make sure you are aware of these. They can and will
be changed to be more flexible in the future. It uses discord's new
[slash commands](https://discord.com/developers/docs/interactions/slash-commands) to perform
the commands, no ! commands. You have an admin/mod role that the bot will operate under. This means it is
your responsibility to ensure the people with access to this channel are trustworthy. Only users under
this role will be allowed to use privileged commands. There is currently no direct RCON command for the
bot. gbans will use RCON internally, but it's not exposed to the bot yet. Once finer access controls get
integrated it will possibly, also be added. All bans, steamid & network, apply to all servers, there is
currently no way to ban on specific servers only. You must enable the `applications.commands` and `bot`
oauth roles.

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

### /ban user_identifier duration \[reason\]

Ban a user from the server

### /banip cidr duration \[reason\]

Ban a network block or IP. To ban a single ip use a /32 block, eg: `10.11.12.100/32`. To ban a
subnet block use a larger mask. eg: `10.11.12.0/24`. This would block all ips between: `10.11.12.1 - 10.11.12.255`

Banning a network:

     user: /banip 1.2.3.0/24 5m Example network ban
    gbans: IP ban created successfully

Banning an IP:

     user: /banip 1.2.3.4/32 5m Example IP ban
    gbans: IP ban created successfully

### /find user_identifier

Locate a player by name or steamid, returning their steamid and current server-id.

     user: /find tim
    gbans: Found player Recliner_Tim (76561198282517317) @ us-1

### /mute user_identifier duration \[reason\]

### /check steamid

Check the current ban state of a steamid

    user: /check 76561199093644873

### /unban steamid

Unbans a previously banned user ahead of the expiration time.

    user: /unban 76561199093644873
    gbans: User ban is now inactive

### /kick user_identifier \[reason\]

Kick a currently playing user from the server

### /psay server-id name/steamid message

Sends a private chat message to a single target. This just calls the [sourcemod command](https://wiki.alliedmods.net/Admin_Commands_(SourceMod))
`sm_psay`

### /csay server-id|* message

Sends a centered message to all players. This just calls the [sourcemod command](https://wiki.alliedmods.net/Admin_Commands_(SourceMod))
`sm_csay`

### /say server-id|* message

Sends a say-chat message to all players.  This just calls the [sourcemod command](https://wiki.alliedmods.net/Admin_Commands_(SourceMod))
`sm_say`
