//go:build !windows

package health

import (
	"errors"
	"fmt"
	"syscall"
)

var errInvalidBlockSize = errors.New("invalid filesystem block size")

// diskStats returns total and available bytes for the filesystem containing root.
func diskStats(root string) (uint64, uint64, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(root, &stats); err != nil {
		return 0, 0, fmt.Errorf("statfs %s: %w", root, err)
	}

	if stats.Bsize < 0 {
		return 0, 0, errInvalidBlockSize
	}

	blockSize := uint64(stats.Bsize) //nolint:gosec // negative values are rejected above

	return stats.Blocks * blockSize, stats.Bavail * blockSize, nil
}
