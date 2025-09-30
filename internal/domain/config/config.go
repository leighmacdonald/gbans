package config

type LinkablePath interface {
	// Path returns the HTTP path that is represented by the instance.
	Path() string
}

type Linker interface {
	ExtURL(obj LinkablePath) string
	ExtURLRaw(path string, args ...any) string
}
