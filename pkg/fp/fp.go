package fp

// Uniq will return a unique list of value from the input list
func Uniq[T comparable](input []T) (uniq []T) {
	if len(input) == 0 {
		return
	}
	uniq = append(uniq, input[0])
	for _, v := range input {
		found := false
		for _, known := range uniq {
			if v == known {
				found = true
				break
			}
		}
		if !found {
			uniq = append(uniq, v)
		}
	}
	return
}
