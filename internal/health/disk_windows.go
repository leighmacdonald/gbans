//go:build windows

package health

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// diskStats returns total and available bytes for the filesystem containing root.
func diskStats(root string) (uint64, uint64, error) {
	path, errPath := windows.UTF16PtrFromString(root)
	if errPath != nil {
		return 0, 0, fmt.Errorf("convert disk path %s: %w", root, errPath)
	}

	var available, total, free uint64
	if err := windows.GetDiskFreeSpaceEx(path, &available, &total, &free); err != nil {
		return 0, 0, fmt.Errorf("get disk free space for %s: %w", root, err)
	}

	return total, available, nil
}
