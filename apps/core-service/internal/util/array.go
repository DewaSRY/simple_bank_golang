package util

// mapSlice applies f to every element of items. Query results that embed a
// shared view/table model (see toAccountuserResponse, toAccountEntriesViewResponse)
// only need this once per response shape, not once per query.
func MapSlice[T, R any](items []T, f func(T) R) []R {
	result := make([]R, len(items))
	for i, item := range items {
		result[i] = f(item)
	}
	return result
}
