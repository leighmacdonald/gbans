package vpksign

import (
	"context"
	"os"
	"os/exec"
	"path"
)

func call(ctx context.Context, vpkBinRoot string, args ...string) (*exec.Cmd, error) {
	bin := path.Join(vpkBinRoot, "vpk_linux32")
	if errEnv := os.Setenv("LD_LIBRARY_PATH", vpkBinRoot); errEnv != nil {
		return nil, errEnv
	}

	return exec.CommandContext(ctx, bin, args...), nil
}

func Sign(ctx context.Context, vpkBinRoot string, inputFilePath string, privateKey string) error {
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
