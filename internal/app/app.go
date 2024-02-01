package app

import (
	"github.com/leighmacdonald/gbans/internal/domain"
)

var (
	BuildVersion = "master" //nolint:gochecknoglobals
	BuildCommit  = ""       //nolint:gochecknoglobals
	BuildDate    = ""       //nolint:gochecknoglobals
)

func Version() domain.BuildInfo {
	return domain.BuildInfo{
		BuildVersion: BuildVersion,
		Commit:       BuildCommit,
		Date:         BuildDate,
	}
}
