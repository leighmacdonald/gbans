package fs

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

// FindFile will walk up the directory tree until it find a file. Max depth of 4 or the minRootDir directory
// is matched.
func FindFile(fileName string, minRootDir string) string {
	var dots []string //nolint:prealloc
	for range 4 {
		dir := path.Join(dots...)
		fPath := path.Join(dir, fileName)

		if Exists(fPath) {
			fp, err := filepath.Abs(fPath)
			if err == nil {
				return fp
			}

			return fp
		}

		if strings.HasSuffix(dir, minRootDir) {
			return fileName
		}

		dots = append(dots, "..")
	}

	return fileName
}
