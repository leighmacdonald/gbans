package util

import (
	"context"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

type WatchFunc func(path string) error

func WatchDir(ctx context.Context, dir string, fn WatchFunc) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.Errorf("Failed to close watcher cleanly: %v", err)
		}
	}()
	if err := watcher.Add(dir); err != nil {
		log.Fatalf("Failed to add watch dir: %v", dir)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				continue
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				if err := fn(event.Name); err != nil {
					log.Errorf("Error executing watcher fn: %v", err)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				continue
			}
			log.Errorf("Watcher error: %v", err)
		}
	}
}
