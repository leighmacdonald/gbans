# Installation Guide

Basic installation overview of the gbans server and sourcemod plugin.

## Sourcemod Plugins

The following extensions must be installed for gbans to work, see their documentation for up to date installation
instructions:

- [System2](https://github.com/dordnung/System2) Provides HTTP(S) client functionality
- [sm-json](https://github.com/clugg/sm-json) `Required for development only` Provides JSON encoding/decoding.
 
## gbans Server

Precompiled binaries will be provided once the project is in a more stable state.

- [make](https://www.gnu.org/software/make/) For predefined build commands
- [golang 1.16+](https://golang.org/) gbans server is written in go.
- [gcc](https://gcc.gnu.org/) Required to build the sqlite3 go extension

Basic steps to build the binary:

    1. git clone git@github.com:leighmacdonald/gbans.git && cd gbans
    2. make
 
You should now have a binary in the project root called `gbans` or `gbans.exe` if on windows.

## Configuration

### Server

Copy the example configuration `gbans_example.yml` and name it `gbans.yml`. It should be in
the same directory as the binary. Configure it as desired. Discord is currently highly recommended, at 
least until the webui is created.

```yaml
http:
  # Listen on this ip address
  # 0.0.0.0 = Any
  host: 0.0.0.0
  # Listen on this port
  port: 6006
  # Run mode for the HTTP service
  # Should normally be "release"
  mode: "release" # release, debug, test

database:
  # Path to your database
  # NOTE: if you are using docker its *VERY* important to change this to /app/database/db.sqlite
  # otherwise your database will be wiped when you pull a new image
  path: "db.sqlite"

discord:
  # Enable optional discord integration
  enabled: false
  # Your discord bot token
  # See: https://discord.com/developers/applications
  token: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  # People in these channels automatically have moderator privilege
  # To find these, Right click the channel -> copy id
  mod_channel_ids:
    - "123456789012345678"
    - "234567890123456789"
  # Commands all start with this character
  prefix: "!"

logging:
  # Set the debug log level
  level: debug
  # Force console colours when it cant guess. This is mostly useful on windows
  force_colours: true
  # Force disable any colouring
  disable_colours: false
  # Show the function + line number where the log message was created
  report_caller: false
  # Show full timestamps in the logs
  full_timestamp: false
```

#### Adding servers

Servers must be registered in the database for them to work. Currently, the only way to automatically
do this is with the `gbans addserver` command. 

`gban addserver <server_name> <addr> <port> <rcon>`

```
./gbans addserver example-1 10.0.0.1 27015 my_rcon_pass
INFO[0000] Using config file: gbans.yaml 
INFO[0000] Added server example-1 with token QecfPbmJeueCrczjetUB 
```

Save the token returned, it will be used in the sourcemod configuration below. Each server
must have a unique token. 

#### Starting the server

To start the server just run `./gbans serve`. It should show output similar to the following if 
successful.

```
➜  gbans git:(master) ✗ ./gbans serve
INFO[0000] Using config file: gbans.yaml 
INFO[0000] Starting gbans service                       
DEBU[0000] Ban sweeper routine started                  
INFO[0000] Bot is now running.  Press CTRL-C to exit.   
INFO[0000] Connected to session ws API                  
```

It's recommended to create a [systemd .service](https://freedesktop.org/software/systemd/man/systemd.service.html) 
file so that it can start automatically. More info on configuring this will be available at a later
date.

### Sourcemod

Place the `sourcemod/plugins/gbans.smx` file into `tf/addons/sourcemod/plugins`. Then add the config as 
described below.

This config file should be places in `tf/addons/sourcemod/configs/gbans.cfg`.

```
"gbans"
{
	// Remote gban server host
	"host"	"https://gbans.example.com"

	// Remote gban server port
	"port"	"443"

	// Unique server name for this server, the same as a "server-id"
	"server_name"	"example-1"

	// The authentication token used to retrieve a auth token
	"server_key"	"YOUR_TOKEN"
}
```

The server gbans server is running you should now be able to see the `[GB]` message logs in the
console. With a message like below on successful authentication with the server.

```
[GB] Using config file: addons/sourcemod/configs/gbans.cfg
[GB] Request to https://gbans.example.com/v1/auth finished with status code 200 in 0.01 seconds
[GB] Successfully authenticated with gbans server
```
