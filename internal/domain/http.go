package domain

func NewLazyResult(count int64, data any) LazyResult {
	if count == 0 {
		return LazyResult{0, []interface{}{}}
	}

	return LazyResult{Count: count, Data: data}
}

type LazyResult struct {
	Count int64 `json:"count"`
	Data  any   `json:"data"`
}
