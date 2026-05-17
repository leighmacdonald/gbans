// Package ptr provides two pointer helper functions.
package ptr

// From returns the dereferenced value if non-nil, otherwise return the default value for the type.
func From[T any](v *T) T {
	if v != nil {
		return *v
	}

	var t T

	return t
}
