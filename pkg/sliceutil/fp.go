// Package sliceutil provides basic generic functions for operating over slices
package sliceutil

import (
	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Integer | constraints.Float
}

// Uniq will return a unique list of value from the input list.
func Uniq[T comparable](input []T) []T { //nolint:ireturn
	var output []T
	if len(input) == 0 {
		return output
	}

	found := make(map[T]bool)
	output = append(output, input[0])

	for _, value := range input {
		if !found[value] {
			found[value] = true
			output = append(output, value)
		}
	}

	return output
}

// FirstPositive returns the first positive value in the slice.
func FirstPositive[T Number](numbers ...T) T { //nolint:ireturn
	for _, curValue := range numbers {
		if curValue > 0 {
			return curValue
		}
	}

	return 0
}
