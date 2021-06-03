package relay

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/web"
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
		log.Fatalf("Invalid log path: %s", path)
		return
	}
	log.Debugf("fileReader starting: %v", path)
	for {
		select {
		case line := <-t.Lines:
			if line == nil {
				continue
			}
			m := strings.TrimRight(line.Text, "\n")
			log.Debugf("Line: %s", m)
			if m == "" {
				continue
			}
			messageChan <- m
		case <-ctx.Done():
			log.Debugf("fileReader shutting down: %v", path)
			return
		}
	}
}

func newFileWatcher(ctx context.Context, directory string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if errW := watcher.Close(); errW != nil {
			log.Errorf("failed to close watcher cleanly: %v", errW)
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
					log.Debugf("Created file: %s", event.Name)
					if !first {
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
				log.Errorf("File watcher error: %v", errW)
			}
		}
	}()

	err = watcher.Add(directory)
	if err != nil {
		log.Fatal(err)
	}
	<-ctx.Done()
}

// New creates and starts a new log reader client instance
func New(ctx context.Context, name string, logPath string, address string, password string) error {
	client, errC := newClient(ctx, name, address, password)
	if errC != nil {
		return errC
	}
	doConnect := func() {
		if !client.isOpen() {
			if err := client.connect(); err != nil {
				log.Errorf("Failed to connect: %v", err)
			}
		}
	}
	go newFileWatcher(ctx, logPath)

	connWatch := time.NewTicker(time.Second * 5)
	doConnect()
	for {
		select {
		case <-connWatch.C:
			doConnect()
		case msg := <-messageChan:
			if !client.isOpen() || !client.authenticated {
				continue
			}
			p, e := web.EncodeWSPayload(web.LogType, web.WebSocketLogPayload{
				ServerName: name,
				Message:    msg,
			})
			if e != nil {
				log.Errorf("Failed to encode ws payload: %v", e)
				continue
			}
			if err := client.enqueue(p); err != nil {
				log.Errorf("Failed to enqueue paylad: %v", err)
			}
		case <-ctx.Done():
			log.Debugf("relay client shutting down")
			return nil
		}
	}
}

func init() {
	messageChan = make(chan string)
}
