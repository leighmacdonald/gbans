package relay

import (
	"bytes"
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"regexp"
	"strings"
)

var (
	reSay = regexp.MustCompile(`"(.+?)<\d+><(\[.+?])>.+?(say|say_team) "(.+?)"$`)
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
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		log.Fatalf("Failed to resolve addr: %v", err)
		return
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("Failed to dial addr: %v", err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close conn: %v", err)
		}
	}()
	messageChan := make(chan string, 5000)
	messageChan <- `L 08/10/2020 - 12:11:04: "BOT<1><[U:0:0]><Red> say "Online"`
	go newFileWatcher(ctx, logPath, messageChan)
	errChan := make(chan error)
	for {
		select {
		case msg := <-messageChan:
			match := reSay.FindStringSubmatch(msg)
			if len(match) != 5 {
				continue
			}
			sid64 := steamid.SID3ToSID64(steamid.SID3(match[2]))
			if sid64.Int64() != 76561197960265728 && !sid64.Valid() {
				continue
			}
			team := false
			if match[3] == "say_team" {
				team = true
			}
			b, err2 := Encode(Payload{
				Type:     TypeLog,
				Server:   name,
				SayTeam:  team,
				Message:  match[4],
				Username: match[1],
				SteamID:  sid64,
			})
			if err2 != nil {
				log.Errorf("Error encoding payload")
				break
			}
			_, err2 = io.Copy(conn, bytes.NewReader(b))
			if err2 != nil {
				log.Errorf("Error writing payload")
				return
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
