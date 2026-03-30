package ptr

// To provides a trivial helper to return a pointer to the value passed in.
// TODO use go.1.26 new()
func To[T any](v T) *T {
	return &v
}
