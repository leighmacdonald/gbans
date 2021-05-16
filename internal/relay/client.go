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
	httpClient  *http.Client
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
func New(ctx context.Context, name string, logPath string, address string, timeout time.Duration) error {
	url := address + "/api/log"
	sendPayload := func(payload []service.LogPayload) error {
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
	go newFileWatcher(ctx, logPath)
	var messageQueue []service.LogPayload
	duration := time.Second * 5
	ticker := time.NewTicker(duration)
	for {
		select {
		case <-ticker.C:
			if len(messageQueue) > 0 {
				log.Debugf("Flushing message queue (timer): len %d", len(messageQueue))
				go func(messages []service.LogPayload) {
					if err := sendPayload(messages); err != nil {
						log.Errorf("Failed to send queued log payload: %v", err)
					}
				}(messageQueue)
				messageQueue = nil
			}
		case msg := <-messageChan:
			messageQueue = append(messageQueue, service.LogPayload{ServerName: name, Message: msg})
			log.Debugf("Added message to log queue: len %d", len(messageQueue))
			if len(messageQueue) >= 25 {
				log.Debugf("Flushing message queue (size): len %d", len(messageQueue))
				go func(messages []service.LogPayload) {
					if err := sendPayload(messages); err != nil {
						log.Errorf("Failed to send queued log payload: %v", err)
					}
				}(messageQueue)
				messageQueue = nil
				ticker.Reset(duration)
			}
		case <-ctx.Done():
			log.Debugf("relay client shutting down")
			return nil
		}
	}
}

func init() {
	messageChan = make(chan string)
	httpClient = &http.Client{Timeout: time.Second * 15}
}
