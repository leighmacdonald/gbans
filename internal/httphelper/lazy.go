package httphelper

type LazyResult struct {
	Count int64 `json:"count"`
	Data  any   `json:"data"`
}

func NewLazyResult(count int64, data any) LazyResult {
	if count == 0 {
		// Return an empty list instead of null
		return LazyResult{0, []any{}}
	}

	return LazyResult{Count: count, Data: data}
}
