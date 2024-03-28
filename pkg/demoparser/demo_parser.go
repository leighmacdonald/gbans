// Package demoparser provides a basic wrapper around https://github.com/demostf/parser
// If the binary does not exist, it will be downloaded to the current directory
package demoparser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/leighmacdonald/gbans/pkg/log"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	binPath     = "parse_demo"
	downloadURL = "https://github.com/demostf/parser/releases/download/v0.4.0/parse_demo"
)

var (
	ErrDecode        = errors.New("failed to decode into parse_demo output json")
	ErrCreateRequest = errors.New("failed to create download request")
	ErrDownload      = errors.New("failed to download parse_demo binary")
	ErrOpenFile      = errors.New("failed to create new fd")
	ErrWrite         = errors.New("failed to write binary")
	ErrCloseBin      = errors.New("failed to close binary file")
	ErrCall          = errors.New("failed to call parser binary")
)

//nolint:tagliatelle
type Player struct {
	Classes map[string]int `json:"classes"` // class -> count?
	Name    string         `json:"name"`
	UserID  int            `json:"userId"`
	SteamID string         `json:"steamId"`
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

//nolint:tagliatelle
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

	for _, user := range d.Users {
		sid64 := steamid.New(user.SteamID)
		if !sid64.Valid() {
			continue
		}

		ids = append(ids, sid64)
	}

	return ids
}

func Parse(ctx context.Context, demoPath string, info *DemoInfo) error {
	if errEnsure := ensureBinary(ctx); errEnsure != nil {
		return errEnsure
	}

	output, errExec := callBin(demoPath)
	if errExec != nil {
		return errExec
	}

	if errDecode := json.NewDecoder(bytes.NewReader(output)).Decode(info); errDecode != nil {
		return errors.Join(errDecode, ErrDecode)
	}

	return nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

func ensureBinary(ctx context.Context) error {
	fullPath := fullBinPath()

	if Exists(fullPath) {
		return nil
	}

	client := http.Client{
		Timeout: time.Second * 60,
	}

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if errReq != nil {
		return errors.Join(errReq, ErrCreateRequest)
	}

	resp, errResp := client.Do(req)
	if errResp != nil {
		return errors.Join(errResp, ErrDownload)
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			slog.Error("failed to close response body", log.ErrAttr(errClose))
		}
	}()

	openFile, err := os.OpenFile(fullPath, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0x755)
	if err != nil {
		return errors.Join(err, ErrOpenFile)
	}

	defer func() {
		if errClose := openFile.Close(); errClose != nil {
			slog.Error("failed to close output file", log.ErrAttr(errClose))
		}
	}()

	if _, errWrite := io.Copy(openFile, resp.Body); errWrite != nil {
		return errors.Join(errWrite, ErrWrite)
	}

	return nil
}

func appDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	fullDir := filepath.Join(dir, ".config", "parse_demo")
	if errMkdir := os.MkdirAll(fullDir, fs.ModePerm); errMkdir != nil {
		panic(errMkdir)
	}

	return fullDir
}

func fullBinPath() string {
	return filepath.Join(appDir(), binPath)
}

func callBin(arg string) ([]byte, error) {
	cmd, errOutput := exec.Command(fullBinPath(), arg).Output() //nolint:gosec
	if errOutput != nil {
		return nil, errors.Join(errOutput, ErrCall)
	}

	return cmd, nil
}
