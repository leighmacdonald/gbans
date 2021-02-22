package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
	"github.com/leighmacdonald/gbans/service"
	log "github.com/sirupsen/logrus"
)

var (
	BuildVersion = "master"

	httpClient *http.Client
)

func fileReader(ctx context.Context, path string, messageChan chan string) {
	t, err := tail.TailFile(path, tail.Config{Follow: true, MaxLineSize: 2000, Poll: true})
	if err != nil {
		log.Fatalf("Invalid log path: %s", path)
		return
	}
	for {
		select {
		case line := <-t.Lines:
			if line == nil {
				continue
			}
			m := strings.TrimRight(line.Text, "\r\n")
			if m == "" {
				continue
			}
			messageChan <- m
		case <-ctx.Done():
			log.Infof("Stopped fileReader: %v", path)
			return
		}
	}
}

func newFileWatcher(ctx context.Context, directory string, newFileChan chan string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.Errorf("failed to close watcher cleanly: %v", err)
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
				//log.Println("event:", event)
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("created file:", event.Name)
					if !first {
						cancel()
					}
					first = false
					c, cancel = context.WithCancel(ctx)
					go fileReader(c, event.Name, newFileChan)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					cancel()
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(directory)
	if err != nil {
		log.Fatal(err)
	}
	<-ctx.Done()
}

func NewClient(ctx context.Context, name string, logPath string, address string) (err error) {
	url := address + "/sapi/v1/log"
	messageChan := make(chan string, 5000)
	messageChan <- `L 08/10/2020 - 12:11:04: "BOT<1><[U:0:0]><Red> say "Online"`
	go newFileWatcher(ctx, logPath, messageChan)
	errChan := make(chan error)
	for {
		select {
		case msg := <-messageChan:
			p := service.RelayPayload{
				Type:    service.TypeLog,
				Server:  name,
				Message: msg,
			}
			b, err := json.Marshal(p)
			if err != nil {
				log.Errorf("Error encoding payload")
				break
			}

			req, err := http.NewRequest("POST", url, bytes.NewReader(b))
			if err != nil {
				log.Errorf("Error encoding payload")
				break
			}
			resp, err := httpClient.Do(req)
			if err != nil {
				log.Errorf("Error encoding payload")
				break
			}
			if resp.StatusCode != http.StatusCreated {
				log.Errorf("Invalid respose received: %s", resp.Status)
			}
		case <-ctx.Done():
			fmt.Println("cancelled")
			err = ctx.Err()
			return
		case err = <-errChan:
			log.Fatalf("Fatal error occurred: %v", err)
			return
		}
	}
}

func init() {
	httpClient = &http.Client{Timeout: time.Second * 5}
}
