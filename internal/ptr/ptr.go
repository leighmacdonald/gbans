package ptr

// To provides a trivial helper to return a pointer to the value passed in.
// TODO use go.1.26 new()
func To[T any](v T) *T {
	return &v
}

func From[T any](v *T) T {
	if v != nil {
		return *v
	}

	var t T

	return t
}
