package ptr

// To provides a trivial helper to return a pointer to the value passed in.
func To[T any](v T) *T {
	return &v
}
