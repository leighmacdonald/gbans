# agent

The agent is, or will soon to be, capable of:

- Console redirection
  - Forwarding stdout/stderr/stdin between remote processes
- Game Installation
  - Download game server files via [DepotDownloader](https://github.com/SteamRE/DepotDownloader)
  - Generate custom per-server configurations.
  - Copy plugins or any other files to the remote game installation
  - Update & Verify game data
- Basic Controls
  - Start
  - Stop
  - Restart

## Topology

Here is a basic topology for game agent deployments.
 
                       [ Master gbans instance ]
                           /               \
                        (ws)            (ws)
                       /                     \
            [ Remote gbans agent ]        [ Remote gbans agent ]
           /       |         |              |        |        \
    [ Game#1 ] [ Game#2] [ Game#3 ]   [ Game#4 ] [ Game#5] [ Game#6 ]
       

## Service

You can create a systemd service with something like this.

`/etc/systemd/system/gbans-agent.service`

```yaml
[Unit]
Description=gbans agent
After=network.target

[Service]
User=steam
Group=steam
WorkingDirectory=/home/steam
ExecStart=/home/steam/gbans agent
Restart=always

[Install]
WantedBy=multi-user.target
```