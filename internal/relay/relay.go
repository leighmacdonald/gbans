// Package relay implements client or agent like functionality to communicate with the central
// gbans instance. Currently it is very simple and only implements a log relaying service.
//
//
// ./gbans relay -H wss://host.com:443 -l ./path/to/tf/logs -n srv-1 -p server_auth_pass
//
package relay

import (
	"context"
	"encoding/json"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/web"
	"github.com/leighmacdonald/gbans/internal/web/client"
	"github.com/leighmacdonald/golib"
	"github.com/pkg/errors"
	"io/ioutil"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
	log "github.com/sirupsen/logrus"
)

var (
	messageChan chan string
)

func fileReader(ctx context.Context, path string) {
	t, err := tail.TailFile(path, tail.Config{Follow: true, MaxLineSize: 2000, Poll: true})
	if err != nil {
		log.WithFields(log.Fields{"filename": path}).Fatal("Invalid log path")
		return
	}
	log.WithFields(log.Fields{"filename": path}).Debug("fileReader starting")
	for {
		select {
		case line := <-t.Lines:
			if line == nil {
				continue
			}
			m := strings.TrimRight(line.Text, "\n")
			if m == "" {
				continue
			}
			messageChan <- m
		case <-ctx.Done():
			log.WithFields(log.Fields{"filename": path}).Debug("fileReader shutting down")
			return
		}
	}
}

func newFileWatcher(ctx context.Context, directory string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithFields(log.Fields{"directory": directory}).Fatalf("Failed to create watcher instance: %v", err)
	}
	defer func() {
		if errW := watcher.Close(); errW != nil {
			log.WithFields(log.Fields{"directory": directory}).Errorf("failed to close watcher cleanly: %v", errW)
		}
	}()
	var (
		cancel context.CancelFunc
		c      context.Context
		first  = true
	)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.WithFields(log.Fields{"filename": event.Name}).Debugf("New file created")
					if !first {
						// If this is not the first file, cancel the existing reader context
						cancel()
					}
					first = false
					c, cancel = context.WithCancel(ctx)
					go fileReader(c, event.Name)
				}
			case errW, ok := <-watcher.Errors:
				if !ok {
					cancel()
					return
				}
				log.WithFields(log.Fields{"directory": directory}).Errorf("File watcher error: %v", errW)
			}
		}
	}()

	err = watcher.Add(directory)
	if err != nil {
		log.WithFields(log.Fields{"directory": directory}).Fatalf("Failed to add directory to watcher: %v", err)
	}
	<-ctx.Done()
}

// New creates and starts a new log reader client instance
func New(ctx context.Context, name string, logPath string, address string, token string) error {
	cli, errC := client.New(ctx, address, name, token)
	if errC != nil {
		return errors.Wrapf(errC, "Failed to create new websocket client")
	}
	defer func() {
		if err := cli.Close(); err != nil {
			cli.Log().Errorf("Error trying to close websocket connection gracefully: %v", err)
		}
	}()
	if config.General.Mode == config.Debug {
		go func() {
			// TODO remove, its only for testing on windows
			// Send test data forever since windows doesnt sync() the log file until
			// after the game has ended.
			f := golib.FindFile("test_data/log_sup_med_1.log", "gbans")
			b, _ := ioutil.ReadFile(f)
			lines := strings.Split(string(b), "\r\n")
			lineNum := 0
			for {
				messageChan <- lines[lineNum]
				lineNum++
				if lineNum >= len(lines) {
					lineNum = 0
				}
				time.Sleep(time.Millisecond * 100)
			}
		}()
	}
	go newFileWatcher(ctx, logPath)
	connWatch := time.NewTicker(time.Second * 5)
	cli.Connect()
	for {
		select {
		case <-connWatch.C:
			cli.Connect()
		case msg := <-messageChan:
			var e error
			switch cli.State() {
			case web.AwaitingAuthentication:
				e = onAuthResp(cli)
				if e == nil {
					cli.Log().Infof("Connection authenticated successfully")
				}
			case web.Authenticated:
				e = onAuthenticatedMessage(cli, name, msg)
			}
			if e != nil {
				cli.Log().Errorf("Error handling message: %v", e)
			}
		case <-ctx.Done():
			cli.Log().Debugf("relay cli shutting down")
			return nil
		}
	}
}

func onAuthResp(cli *client.Client) error {
	var resp web.SocketPayload
	if errResp := cli.ReadJSON(&resp); errResp != nil {
		return errors.Wrapf(errResp, "Failed to read authentication reply: %v", errResp)
	}
	var authResp web.WebSocketAuthResp
	if errAuthResp := json.Unmarshal(resp.Data, &authResp); errAuthResp != nil {
		return errors.Wrapf(errAuthResp, "Failed to read authentication payload: %v", errAuthResp)
	}
	if !authResp.Status {
		return errors.New("Authentication status failed")
	}
	cli.SetState(web.Authenticated)
	return nil
}

func onAuthenticatedMessage(cli *client.Client, name string, msg string) error {
	p, e := web.EncodeWSPayload(web.LogType, web.SocketLogPayload{
		ServerName: name,
		Message:    msg,
	})
	if e != nil {
		return errors.Wrapf(e, "Failed to encode ws payload")
	}
	cli.Enqueue(p)
	cli.Log().Debugf("Relayed: %s", msg)
	return nil
}

func init() {
	messageChan = make(chan string)
}
