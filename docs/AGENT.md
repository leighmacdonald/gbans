# agent

The agent is, or will soon to be, capable of:

- [x] UDP Listener to receive logs from game servers 
  - [x] `logaddress_add` is used to send log lines in real-time over the local network, avoiding writing logs to local disk.
  - [ ] `sv_logsecret` To identify the server sending the message.
  - [ ] Forward message to central server
- [ ] Console redirection
  - [ ] Forwarding stdout/stderr/stdin between remote processes
- [ ] Game Installation
  - [x] Download game server files via [DepotDownloader](https://github.com/SteamRE/DepotDownloader)
  - [ ] Generate custom per-server configurations.
  - [ ] Copy plugins or any other files to the remote game installation
  - [ ] Update & Verify game data
- [ ] Basic Controls
  - [ ] Start
  - [ ] Stop
  - [ ] Restart
  
The agent will buffer 50 messages by default. Further messages are ignored.

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