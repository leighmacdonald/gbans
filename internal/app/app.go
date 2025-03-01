package app

var (
	BuildVersion = "master" //nolint:gochecknoglobals
	BuildCommit  = ""       //nolint:gochecknoglobals
	BuildDate    = ""       //nolint:gochecknoglobals
	SentryDSN    = ""       //nolint:gochecknoglobals
)

type BuildInfo struct {
	BuildVersion string
	Commit       string
	Date         string
}

func Version() BuildInfo {
	return BuildInfo{
		BuildVersion: BuildVersion,
		Commit:       BuildCommit,
		Date:         BuildDate,
	}
}
