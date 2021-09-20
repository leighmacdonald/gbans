package agent

import (
	"bytes"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path"
)

type GameInstance interface {
	Send(msg []byte)
	Start() error
	Stop() error
	Attach()
	Update() error
}

// SourceDS handles starting/stopping the actual game processes
type SourceDS struct {
	appId    steamid.AppID
	gameDir  string
	game     string
	template string
	cmd      *exec.Cmd
	l        *log.Entry
}

func NewSourceDS(gameDir string, appId steamid.AppID) (*SourceDS, error) {
	execPath := path.Join(gameDir, "srcds_run")
	if !golib.Exists(execPath) {
		return nil, errors.New("Invalid gameDir, srcds_run not found")
	}
	srcds := &SourceDS{
		appId:    appId,
		gameDir:  gameDir,
		template: "default",
		cmd:      exec.Command(execPath),
		l:        log.WithFields(log.Fields{"app": appId, "dir": gameDir, "template": "default"}),
	}
	addEnv := "FOO=bar"
	srcds.cmd.Env = append(os.Environ(), addEnv)

	var stdoutBuf, stderrBuf bytes.Buffer
	srcds.cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	srcds.cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	return srcds, nil
}

func (srcds SourceDS) Start() error {
	srcds.l.Info("Starting srcds instance")
	return srcds.cmd.Start()
}

func (srcds SourceDS) Stop() error {
	srcds.l.Info("Stopping srcds instance")
	return srcds.cmd.Process.Signal(os.Kill)
}

func (srcds SourceDS) Send(msg []byte) error {
	return nil
}

func (srcds SourceDS) Attach() {

}

func (srcds SourceDS) Update() error {
	return nil
}
