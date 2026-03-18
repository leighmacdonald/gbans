package httphelper

// LazyResult represents a lazy-loaded database result with included total result count data.
type LazyResult[T any] struct {
	// Count of -1 denotes unknown total count
	Count int64 `json:"count"`
	Data  []T   `json:"data"`
}

func NewLazyResult[T any](count int64, data []T) LazyResult[T] {
	if count == 0 {
		// Return an empty list instead of null
		return LazyResult[T]{0, []T{}}
	}

	return LazyResult[T]{Count: count, Data: data}
}
