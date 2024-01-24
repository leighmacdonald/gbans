package vpksign

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
)

var ErrLibraryPath = errors.New("failed to set LD_LIBRARY_PATH")

func call(ctx context.Context, vpkBinRoot string, args ...string) (*exec.Cmd, error) {
	bin := path.Join(vpkBinRoot, "vpk_linux32")

	if errEnv := os.Setenv("LD_LIBRARY_PATH", vpkBinRoot); errEnv != nil {
		return nil, errors.Join(errEnv, ErrLibraryPath)
	}

	return exec.CommandContext(ctx, bin, args...), nil
}

func Sign(ctx context.Context, vpkBinRoot string, _ string, privateKey string) error {
	cmd, errCmd := call(ctx, vpkBinRoot, "-k", privateKey)
	if errCmd != nil {
		return errCmd
	}

	_, errOut := cmd.CombinedOutput()
	if errOut != nil {
		return errCmd
	}

	return nil
}
