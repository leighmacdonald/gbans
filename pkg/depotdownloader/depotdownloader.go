package depotdownloader

import (
	"bytes"
	"fmt"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path"
)

const (
	tf2server steamid.AppID = 232250
)

// DepotDownloader provides a simple wrapper around https://github.com/SteamRE/DepotDownloader
// since as far as im aware there is no native solution for downloading the depots available
// currently.
type DepotDownloader struct {
	AppID        steamid.AppID
	InstallPath  string
	DotNetExec   string
	DepotDLL     string
	validate     bool
	maxServers   int
	maxDownloads int
}

const depotDLLName = "DepotDownloader.dll"

// New sets up a DepotDownloader for the specified game.
// It currently expects the executable to be found on the PATH
func New(appID steamid.AppID, installPath string) (*DepotDownloader, error) {
	depotPath, found := os.LookupEnv("DEPOT_DOWNLOADER")
	if !found {
		return nil, errors.New("Must set DEPOT_DOWNLOADER env var to directory containing")
	}
	depotFullPath := path.Join(depotPath, depotDLLName)
	if !golib.Exists(depotPath) {
		return nil, errors.Errorf("%s was not found within %s dir", depotDLLName, depotPath)
	}
	executable := "dotnet"
	p, err := exec.LookPath(executable)
	if err != nil {
		return nil, errors.New("Failed to located dotnet executable")
	}
	log.Debugf("DepotDownloader executable is %s", p)
	return &DepotDownloader{
		AppID:       appID,
		InstallPath: installPath,
		DotNetExec:  p,
		validate:    true,
		DepotDLL:    depotFullPath,
	}, nil
}

// Start performs the actual downloading
func (d DepotDownloader) Start() error {
	args := []string{
		d.DepotDLL,
		"-app", fmt.Sprintf("%d", d.AppID),
		"-dir", d.InstallPath,
		"-max-downloads", fmt.Sprintf("%d", d.maxDownloads),
		"-max-servers", fmt.Sprintf("%d", d.maxServers),
		"-language", "english",
	}
	if d.validate {
		args = append(args, "-validate")
	}
	cmd := exec.Command(d.DotNetExec, args...)
	addEnv := "FOO=bar"
	cmd.Env = append(os.Environ(), addEnv)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Run()
	if err != nil {
		return err
	}
	//outStr, errStr := string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())
	return nil
}
