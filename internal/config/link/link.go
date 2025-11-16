package link

var BaseURL = "http://localhost:6006"

type Linkable interface {
	// Path returns the HTTP path that is represented by the instance.
	Path() string
}

func Path(linkable Linkable) string {
	return BaseURL + linkable.Path()
}

func Raw(path string) string {
	return BaseURL + path
}
