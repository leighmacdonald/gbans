package link

import "sync"

var (
	baseURLMu sync.RWMutex              //nolint:gochecknoglobals
	baseURL   = "http://localhost:6006" //nolint:gochecknoglobals
)

// SetBaseURL updates the global base URL used to build absolute links.
func SetBaseURL(url string) {
	baseURLMu.Lock()
	baseURL = url
	baseURLMu.Unlock()
}

type Linkable interface {
	// Path returns the HTTP path that is represented by the instance.
	Path() string
}

func Path(linkable Linkable) string {
	baseURLMu.RLock()
	defer baseURLMu.RUnlock()

	return baseURL + linkable.Path()
}

func Raw(path string) string {
	baseURLMu.RLock()
	defer baseURLMu.RUnlock()

	return baseURL + path
}
