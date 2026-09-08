package util

//go:fix inline
func ToPtr[T any](s T) *T {
	return new(s)
}
