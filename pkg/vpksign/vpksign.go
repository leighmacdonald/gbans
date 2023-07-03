package vpksign

import (
	"context"
	"os"
	"os/exec"
	"path"

	"github.com/pkg/errors"
)

func call(ctx context.Context, vpkBinRoot string, args ...string) (*exec.Cmd, error) {
	bin := path.Join(vpkBinRoot, "vpk_linux32")

	if errEnv := os.Setenv("LD_LIBRARY_PATH", vpkBinRoot); errEnv != nil {
		return nil, errors.Wrap(errEnv, "Failed to set LD_LIBRARY_PATH")
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
