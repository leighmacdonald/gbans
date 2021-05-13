# relay

gbans comes with a tool that allows server administrators to store, index and query log data. This is accomplished
by a companion agent/process running on all hosts using the relay subcommand `gbans relay`. This tool watches for new 
log files and begins watching them for new data (`tail -f`). These entries are sent over the HTTP api to the central
gbans instance for storage.

## Usage

The relay client command only requires 3 options: host, logdir & name.
    
```Usage:
  gbans relay [flags]

Flags:
  -h, --help             help for relay
  -H, --host string      Server host to send logs to (default "localhost")
  -l, --logdir string    Path to tf2 logs directory
  -n, --name string      Server ID used for identification
  -t, --timeout string   API Timeout (eg: 1s, 1m, 1h) (default "5s")

Global Flags:
      --config string   config file (default is $HOME/.gbans.yaml)
```    

    `$ ./gbans relay -n "us-3" -H https://gbans.server.com:443 -l /home/tf2server/serverfiles/tf/logs`

Note that it will not start monitoring / sending anything until a mapchange occurs, creating a new logfile event.

## Service

You can create a systemd service with something like this.

`/etc/systemd/system/gbans-relay.service`

```yaml
[Unit]
Description=gbans log relay
After=network.target

[Service]
User=tf2server
Group=tf2
WorkingDirectory=/home/tf2server
ExecStart=/home/tf2server/gbans/gbans relay -n "us-3" -H https://gbans.uncledane.com:443 -l /home/tf2server/serverfiles/tf/logs
Restart=always

[Install]
WantedBy=multi-user.target
```