package fp

// Uniq will return a unique list of value from the input list
func Uniq[T comparable](input []T) (output []T) {
	if len(input) == 0 {
		return
	}
	output = append(output, input[0])
	for _, v := range input {
		found := false
		for _, known := range output {
			if v == known {
				found = true
				break
			}
		}
		if !found {
			output = append(output, v)
		}
	}
	return
}

func Contains[T comparable](input []T, value T) bool {
	for _, w := range input {
		if w == value {
			return true
		}
	}
	return false
}
