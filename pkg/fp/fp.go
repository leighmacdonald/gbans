// Package fp provides basic generic functional style functions
package fp

import "golang.org/x/exp/constraints"

type Number interface {
	constraints.Integer | constraints.Float
}

// Uniq will return a unique list of value from the input list.
func Uniq[T comparable](input []T) []T {
	var output []T
	if len(input) == 0 {
		return output
	}

	output = append(output, input[0])

	for _, value := range input {
		found := false

		for _, known := range output {
			if value == known {
				found = true

				break
			}
		}

		if !found {
			output = append(output, value)
		}
	}

	return output
}

func Contains[T comparable](input []T, value T) bool {
	for _, child := range input {
		if child == value {
			return true
		}
	}

	return false
}

func Remove[T comparable](input []T, value T) []T {
	var newValues []T //nolint:prealloc

	for _, existingValue := range input {
		if value == existingValue {
			continue
		}

		newValues = append(newValues, existingValue)
	}

	return newValues
}

func Prepend[T comparable](input []T, value T) []T {
	return append([]T{value}, input...)
}

//nolint:ireturn
func Avg[T Number](numbers []T) T {
	var (
		sum   T
		count T
	)

	for _, curValue := range numbers {
		sum += curValue
		count++
	}

	return sum / count
}

//nolint:ireturn
func Reverse[S ~[]E, E any](s S) S {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}

	return s
}
