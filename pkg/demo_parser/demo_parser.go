// Package demo_parser provides a basic wrapper around https://github.com/demostf/parser
// If the binary does not exist, it will be downloaded to the current directory
package demo_parser

import (
	"bytes"
	"encoding/json"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	binPath     = "parse_demo"
	downloadURL = "https://github.com/demostf/parser/releases/download/v0.4.0/parse_demo"
)

type Player struct {
	Classes map[string]int `json:"classes"` // class -> count?
	Name    string         `json:"name"`
	UserId  int            `json:"userId"`
	SteamId string         `json:"steamId"`
	Team    string         `json:"team"`
}

type Message struct {
	Kind string `json:"kind"`
	From string `json:"from"`
	Text string `json:"text"`
	Tick int    `json:"tick"`
}
type Death struct {
	Weapon   string `json:"weapon"`
	Victim   int    `json:"victim"`
	Assister *int   `json:"assister"`
	Killer   int    `json:"killer"`
	Tick     int    `json:"tick"`
}

type Round struct {
	Winner  string  `json:"winner"`
	Length  float64 `json:"length"`
	EndTick int     `json:"end_tick"`
}

type DemoInfo struct {
	Chat            []Message         `json:"chat"`
	Users           map[string]Player `json:"users"` // userid -> player
	Deaths          []Death           `json:"deaths"`
	Rounds          []Round           `json:"rounds"`
	StartTick       int               `json:"startTick"`
	IntervalPerTick float64           `json:"intervalPerTick"`
}

func (d DemoInfo) SteamIDs() steamid.Collection {
	var ids steamid.Collection

	for key := range d.Users {
		sid64 := steamid.New(key)
		if !sid64.Valid() {
			continue
		}

		ids = append(ids, sid64)
	}

	return ids
}

func Parse(demoPath string, info *DemoInfo) error {
	if errEnsure := ensureBinaryExists(); errEnsure != nil {
		return errEnsure
	}

	output, errExec := callBin(demoPath)
	if errExec != nil {
		return errExec
	}

	if errDecode := json.NewDecoder(bytes.NewReader(output)).Decode(info); errDecode != nil {
		return errors.Wrap(errDecode, "Failed to decode parse_demo output json")
	}

	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ensureBinaryExists() error {
	fullPath := fullBinPath()

	if exists(fullPath) {
		return nil
	}

	client := http.Client{
		Timeout: time.Second * 60,
	}

	resp, errResp := client.Get(downloadURL)
	if errResp != nil {
		return errors.Wrap(errResp, "Failed to download parse_demo binary")
	}

	data, _ := io.ReadAll(resp.Body)

	_ = resp.Body.Close()

	fd, err := os.OpenFile(fullPath, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0x775)
	if err != nil {
		return errors.Wrap(err, "Failed to create new fd")
	}

	if _, errWrite := fd.Write(data); errWrite != nil {
		return errors.Wrap(errWrite, "Failed to write binary")
	}

	if errClose := fd.Close(); errClose != nil {
		return errors.Wrap(errClose, "Failed to close binary fine")
	}

	return nil
}

func fullBinPath() string {
	dir, _ := os.Getwd()

	return filepath.Join(dir, binPath)
}

func callBin(arg string) ([]byte, error) {
	cmd, errOutput := exec.Command(fullBinPath(), arg).Output()
	if errOutput != nil {
		return nil, errors.Wrap(errOutput, "Failed to call parser binary")
	}

	return cmd, nil
}
