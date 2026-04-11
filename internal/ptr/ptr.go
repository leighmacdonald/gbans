// Package ptr provides two pointer helper functions.
package ptr

// To provides a trivial helper to return a pointer to the value passed in.
// TODO use go.1.26 new() .
func To[T any](v T) *T {
	return &v
}

// From returns the value if non-nil, otherwise return the default value for the type.
func From[T any](v *T) T {
	if v != nil {
		return *v
	}

	var t T

	return t
}
