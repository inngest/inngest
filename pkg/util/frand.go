package util

import (
	"fmt"
	"lukechampine.com/frand"
	"math"
	"sync"
)

var (
	ErrWeightedSampleRead = fmt.Errorf("error reading from weighted sample")
)

// FrandRNG is a fast crypto-secure prng which uses a mutex to guard
// parallel reads.  It also implements the x/exp/rand.Source interface
// by adding a Seed() method which does nothing.
type FrandRNG struct {
	*frand.RNG
	lock *sync.Mutex
}

func NewFrandRNG() *FrandRNG {
	return &FrandRNG{RNG: frand.New(), lock: &sync.Mutex{}}
}

func (f *FrandRNG) Read(b []byte) (int, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.RNG.Read(b)
}

func (f *FrandRNG) Uint64() uint64 {
	return f.Uint64n(math.MaxUint64)
}

func (f *FrandRNG) Uint64n(n uint64) uint64 {
	// sampled.Take calls Uint64n, which must be guarded by a lock in order
	// to be thread-safe.
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.RNG.Uint64n(n)
}

func (f *FrandRNG) Float64() float64 {
	// sampled.Take also calls Float64, which must be guarded by a lock in order
	// to be thread-safe.
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.RNG.Float64()
}

func (f *FrandRNG) Seed(seed uint64) {
	// Do nothing.
}
