package faker

import (
	"io"
	mathrand "math/rand"
	"sync"
)

var (
	rand   *mathrand.Rand
	crypto io.Reader
)

type safeSource struct {
	mx sync.Mutex
	mathrand.Source
}

func (s *safeSource) Int63() int64 {
	s.mx.Lock()
	defer s.mx.Unlock()

	return s.Source.Int63()
}

// NewSafeSource wraps an unsafe rand.Source with a mutex to guard the random source
// against concurrent access.
func NewSafeSource(in mathrand.Source) mathrand.Source {
	return &safeSource{
		Source: in,
	}
}

// SetRandomSource sets a new random source at the package level.
//
// To use a concurrent-safe source, you may wrap it with NewSafeSource,
// e.g. SetRandomSource(NewSafeSource(mysource)).
//
// The default is the global, concurrent-safe source provided by math/rand.
func SetRandomSource(in mathrand.Source) {
	rand = mathrand.New(in)
}

// SetCryptoSource sets a new reader for functions using a cryptographically-safe random generator (e.g. UUID).
//
// The default is the global source provided by crypto/rand.
func SetCryptoSource(in io.Reader) {
	crypto = in
}
