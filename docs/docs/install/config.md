---
sidebar_position: 2
---

# Base Configuration

## Server

Copy the example configuration `gbans_example.yml` and name it `gbans.yml`. It should be in
the same directory as the binary. Configure it as desired.

This is your core configuration file. These values cannot be changed without changing this file and restarting
the service. Most other configuration is handled via the webapp admin interface.

```yaml
owner: 76561197960287930
external_url: "http://example.com"

# Listen on this ip address
# 0.0.0.0 = Any
http_host: 0.0.0.0
# Listen on this port
http_port: 6006
http_static_path:
http_client_timeout: 20
# Encryption key for cookies
http_cookie_key: change-me
http_cors_origins:
  - "https://example.com"

# DSN to your database
database_dsn: "postgresql://gbans:gbans@192.168.0.200:5432/gbans"
database_log_queries: false
```

## systemd service

If you are not using docker, it's recommended to create a [systemd .service](https://freedesktop.org/software/systemd/man/systemd.service.html)
file so that it can start automatically. More info on configuring this will be available at a later
date.

## Sourcemod

Place the `sourcemod/plugins/gbans.smx` file into `tf/addons/sourcemod/plugins`. Then add the config as
described below.

This config file should be places in `tf/addons/sourcemod/configs/gbans.cfg`.

```
"gbans"
{
	// Remote gban server host
	"host"	"https://example.com"

	// Remote gban server port
	"port"	"443"

	// Unique server name for this server, the same as a "server-id"
	"server_name"	"example-1"

	// The authentication token used to retrieve a auth token
	"server_key"	"YOUR_SERVER_PASSWORD"
}
```

The server gbans server is running you should now be able to see the `[GB]` message logs in the
console. With a message like below on successful authentication with the server.

## Discord

To use discord you need to [create a discord application](https://discord.com/developers/applications). You will need
the following values from your application:

- Application ID (General -> Application ID)
- Token  (Bot -> Token)
- Client Secret (OAuth2 -> Client Secret)

You will also need to fetch the following ids from your discord client. You will want to enable discord developer mode
to be able to easily acquire these role and channel ids through your own discord client.

- Your main server guild id.
- Logging Channels IDs
  - Log Channel (default catch-all if no others are configured)
  - Match logs
  - Vote logs
  - Appeal / Report logs
  - Ban logs
  - Forum logs
  - Word filter logs

Care should be taken to restrict these channels to permissions as appropriate.

To enable discord connections, You must also set an oauth2 redirect (Oauth2 -> Redirects -> Add) to point to your own server.

    https://example.com/discord/oauth

## IP2Location

To install the GeoLite2 databases, create an account on [IP2location Lite](https://lite.ip2location.com). After
confirmation, you'll be given a download token for use in gbans.yaml.

### Via Web

You can find a button to initiate a refresh of the database via the geo database admin gui at `https://example.com/admin/settings?section=geo_location`.

### Via cli

If instead you want to do it manually, or though cron via CLI you can try the following.

- If using Docker, start the process via `docker exec -it gbans ./gbans net update`.
- If using a compiled binary, navigate to the folder and run `./gbans net update` to start the process.

The process will take up to 30 minutes, depending on hardware, and will add around 2GB to the database when all's said
and done.

## Enabling User Location

The Servers page lets users sort by range. Gbans does not use the locations API to get data from the browser.
Instead, you're required to use Cloudflare to get the location. Gbans must be proxied through Cloudflare to
accomplish this, and setting that up is out of scope of this doc.

Once the domain is set up, go to the domain settings, the `Rules` dropdown, `Transform Rules`, and then the
`Managed Transforms` tab. Enable `Add visitor location headers`, and wait around 5 minutes for it to take effect.
You should then be able to see your location (more or less) on the servers page.
