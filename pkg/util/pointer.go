package util

func ToPtr[T any](s T) *T {
	return &s
}
