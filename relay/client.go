package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"net/http"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
	"github.com/leighmacdonald/gbans/internal/service"
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
			log.Debugf("fileReader shutting down: %v", path)
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
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("created file:", event.Name)
					if !first {
						cancel()
					}
					first = false
					c, cancel = context.WithCancel(ctx)
					go fileReader(c, event.Name, newFileChan)
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

func NewClient(ctx context.Context, name string, logPath string, address string, timeout time.Duration) error {
	url := address + "/api/log"
	sendPayload := func(payload service.LogPayload) error {
		c, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		b, err1 := json.Marshal(payload)
		if err1 != nil {
			return errors.Wrapf(err1, "Error encoding payload")
		}
		req, err2 := http.NewRequestWithContext(c, "POST", url, bytes.NewReader(b))
		if err2 != nil {
			return errors.Wrapf(err2, "Error creating request payload")
		}
		resp, err3 := httpClient.Do(req)
		if err3 != nil {
			return errors.Wrapf(err3, "Error performing request")
		}
		if resp.StatusCode != http.StatusCreated {
			return errors.Errorf("Invalid respose received: %s", resp.Status)
		}
		return nil
	}

	messageChan := make(chan string, 5000)
	go newFileWatcher(ctx, logPath, messageChan)
	for {
		select {
		case msg := <-messageChan:
			if err := sendPayload(service.LogPayload{ServerName: name, Message: msg}); err != nil {
				log.Errorf(err.Error())
			}
		case <-ctx.Done():
			log.Debugf("relay client shutting down")
			return nil
		}
	}
}

func init() {
	httpClient = &http.Client{Timeout: time.Second * 5}
}
