package fp

func Max[T Number](numbers ...T) T { //nolint:ireturn
	var largest T

	for _, number := range numbers {
		if number > largest {
			largest = number
		}
	}

	return largest
}

func Clamp[T Number](number T, min T, max T) T { //nolint:ireturn
	if number > max {
		return max
	}

	if number < min {
		return min
	}

	return number
}
