package util

type Future[T any] struct {
	Value chan T
}

func (f *Future[T]) Get() (T, bool) {
	select {
	case v := <-f.Value:
		return v, true
	default:
		var zero T
		return zero, false
	}
}

type Result[T any] struct {
	Ok  Future[T]
	Err error
}
