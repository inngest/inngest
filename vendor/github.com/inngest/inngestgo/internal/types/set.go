package types

import "sync"

// Set is a generic set, safe for concurrent use.
type Set[T comparable] struct {
	sync.Mutex

	elems map[T]struct{}
}

func (s *Set[T]) Add(v ...T) {
	s.Lock()
	defer s.Unlock()

	if s.elems == nil {
		s.elems = make(map[T]struct{})
	}
	for _, v := range v {
		s.elems[v] = struct{}{}
	}
}

func (s *Set[T]) Contains(v T) bool {
	s.Lock()
	defer s.Unlock()

	_, ok := s.elems[v]
	return ok
}

func (s *Set[T]) Len() int {
	s.Lock()
	defer s.Unlock()

	return len(s.elems)
}

func (s *Set[T]) Remove(v T) {
	s.Lock()
	defer s.Unlock()

	delete(s.elems, v)
}

func (s *Set[T]) ToSlice() []T {
	s.Lock()
	defer s.Unlock()

	slice := make([]T, 0, len(s.elems))
	for v := range s.elems {
		slice = append(slice, v)
	}
	return slice
}
