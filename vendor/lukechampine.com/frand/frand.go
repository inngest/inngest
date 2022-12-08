package frand // import "lukechampine.com/frand"

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	"math/big"
	"strconv"
	"sync"

	"github.com/aead/chacha20/chacha"
)

func erase(b []byte) {
	// compiles to memclr
	for i := range b {
		b[i] = 0
	}
}

func copyAndErase(dst, src []byte) int {
	n := copy(dst, src)
	erase(src[:n])
	return n
}

// An RNG is a cryptographically-strong RNG constructed from the ChaCha stream
// cipher.
type RNG struct {
	buf    []byte
	n      int
	rounds int
}

// Read fills b with random data. It always returns len(b), nil.
//
// For performance reasons, calling Read once on a "large" buffer (larger than
// the RNG's internal buffer) will produce different output than calling Read
// multiple times on smaller buffers. If deterministic output is required,
// clients should call Read in a loop; when copying to an io.Writer, use
// io.CopyBuffer instead of io.Copy. Callers should also be aware that b is
// xored with random data, not directly overwritten; this means that the new
// contents of b depend on its previous contents.
func (r *RNG) Read(b []byte) (int, error) {
	if len(b) <= len(r.buf[r.n:]) {
		// can fill b entirely from buffer
		r.n += copyAndErase(b, r.buf[r.n:])
	} else if len(b) <= len(r.buf[r.n:])+len(r.buf[chacha.KeySize:]) {
		// b is larger than current buffer, but can be filled after a reseed
		n := copy(b, r.buf[r.n:])
		chacha.XORKeyStream(r.buf, r.buf, make([]byte, chacha.NonceSize), r.buf[:chacha.KeySize], r.rounds)
		r.n = chacha.KeySize + copyAndErase(b[n:], r.buf[chacha.KeySize:])
	} else {
		// filling b would require multiple reseeds; instead, generate a
		// temporary key, then write directly into b using that key
		tmpKey := make([]byte, chacha.KeySize)
		r.Read(tmpKey)
		chacha.XORKeyStream(b, b, make([]byte, chacha.NonceSize), tmpKey, r.rounds)
		erase(tmpKey)
	}
	return len(b), nil
}

// Bytes is a helper function that allocates and returns n bytes of random data.
func (r *RNG) Bytes(n int) []byte {
	b := make([]byte, n)
	r.Read(b)
	return b
}

// Entropy256 is a helper function that returns 256 bits of random data.
func (r *RNG) Entropy256() (entropy [32]byte) {
	r.Read(entropy[:])
	return
}

// Entropy192 is a helper function that returns 192 bits of random data.
func (r *RNG) Entropy192() (entropy [24]byte) {
	r.Read(entropy[:])
	return
}

// Entropy128 is a helper function that returns 128 bits of random data.
func (r *RNG) Entropy128() (entropy [16]byte) {
	r.Read(entropy[:])
	return
}

// Uint64n returns a uniform random uint64 in [0,n). It panics if n == 0.
func (r *RNG) Uint64n(n uint64) uint64 {
	if n == 0 {
		panic("frand: argument to Uint64n is 0")
	}
	// To eliminate modulo bias, keep selecting at random until we fall within
	// a range that is evenly divisible by n.
	// NOTE: since n is at most math.MaxUint64, max is minimized when:
	//    n = math.MaxUint64/2 + 1 -> max = math.MaxUint64 - math.MaxUint64/2
	// This gives an expected 2 tries before choosing a value < max.
	max := math.MaxUint64 - math.MaxUint64%n
	b := make([]byte, 8)
again:
	r.Read(b)
	i := binary.LittleEndian.Uint64(b)
	if i >= max {
		goto again
	}
	return i % n
}

// Intn returns a uniform random int in [0,n). It panics if n <= 0.
func (r *RNG) Intn(n int) int {
	if n <= 0 {
		panic("frand: argument to Intn is <= 0: " + strconv.Itoa(n))
	}
	// NOTE: since n is at most math.MaxUint64/2, max is minimized when:
	//    n = math.MaxUint64/4 + 1 -> max = math.MaxUint64 - math.MaxUint64/4
	// This gives an expected 1.333 tries before choosing a value < max.
	return int(r.Uint64n(uint64(n)))
}

// BigIntn returns a uniform random *big.Int in [0,n). It panics if n <= 0.
func (r *RNG) BigIntn(n *big.Int) *big.Int {
	i, _ := rand.Int(r, n)
	return i
}

// Float64 returns a random float64 in [0,1).
func (r *RNG) Float64() float64 {
	return float64(r.Uint64n(1<<53)) / (1 << 53)
}

// Perm returns a random permutation of the integers [0,n). It panics if n < 0.
func (r *RNG) Perm(n int) []int {
	m := make([]int, n)
	for i := 1; i < n; i++ {
		j := r.Intn(i + 1)
		m[i] = m[j]
		m[j] = i
	}
	return m
}

// Shuffle randomly permutes n elements by repeatedly calling swap in the range
// [0,n). It panics if n < 0.
func (r *RNG) Shuffle(n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		swap(i, r.Intn(i+1))
	}
}

// NewCustom returns a new RNG instance seeded with the provided entropy and
// using the specified buffer size and number of ChaCha rounds. It panics if
// len(seed) != 32, bufsize < 32, or rounds != 8, 12 or 20.
func NewCustom(seed []byte, bufsize int, rounds int) *RNG {
	if len(seed) != chacha.KeySize {
		panic("frand: invalid seed size")
	} else if bufsize < chacha.KeySize {
		panic("frand: bufsize must be at least 32")
	} else if !(rounds == 8 || rounds == 12 || rounds == 20) {
		panic("frand: rounds must be 8, 12, or 20")
	}
	buf := make([]byte, chacha.KeySize+bufsize)
	chacha.XORKeyStream(buf, buf, make([]byte, chacha.NonceSize), seed, rounds)
	return &RNG{
		buf:    buf,
		n:      chacha.KeySize,
		rounds: rounds,
	}
}

// "master" RNG, seeded from crypto/rand; RNGs returned by New derive their seed
// from this RNG. This means we only ever need to read system entropy a single
// time, at startup.
var masterRNG = func() *RNG {
	seed := make([]byte, chacha.KeySize)
	n, err := rand.Read(seed)
	if err != nil || n != len(seed) {
		panic("not enough system entropy to seed master RNG")
	}
	return NewCustom(seed, 1024, 12)
}()
var masterMu sync.Mutex

// New returns a new RNG instance. The instance is seeded with entropy from
// crypto/rand, albeit indirectly; its seed is generated by a global "master"
// RNG, which itself is seeded from crypto/rand. This means the frand package
// only reads system entropy once, at startup.
func New() *RNG {
	masterMu.Lock()
	defer masterMu.Unlock()
	return NewCustom(masterRNG.Bytes(32), 1024, 12)
}

// Global versions of each RNG method, leveraging a pool of RNGs.

var rngpool = sync.Pool{
	New: func() interface{} {
		return New()
	},
}

// Read fills b with random data. It always returns len(b), nil.
func Read(b []byte) (int, error) {
	r := rngpool.Get().(*RNG)
	defer rngpool.Put(r)
	return r.Read(b)
}

// Bytes is a helper function that allocates and returns n bytes of random data.
func Bytes(n int) []byte {
	r := rngpool.Get().(*RNG)
	defer rngpool.Put(r)
	return r.Bytes(n)
}

// Entropy256 is a helper function that returns 256 bits of random data.
func Entropy256() [32]byte {
	r := rngpool.Get().(*RNG)
	defer rngpool.Put(r)
	return r.Entropy256()
}

// Entropy192 is a helper function that returns 192 bits of random data.
func Entropy192() [24]byte {
	r := rngpool.Get().(*RNG)
	defer rngpool.Put(r)
	return r.Entropy192()
}

// Entropy128 is a helper function that returns 128 bits of random data.
func Entropy128() [16]byte {
	r := rngpool.Get().(*RNG)
	defer rngpool.Put(r)
	return r.Entropy128()
}

// Uint64n returns a uniform random uint64 in [0,n). It panics if n == 0.
func Uint64n(n uint64) uint64 {
	r := rngpool.Get().(*RNG)
	defer rngpool.Put(r)
	return r.Uint64n(n)
}

// Intn returns a uniform random int in [0,n). It panics if n <= 0.
func Intn(n int) int {
	r := rngpool.Get().(*RNG)
	defer rngpool.Put(r)
	return r.Intn(n)
}

// BigIntn returns a uniform random *big.Int in [0,n). It panics if n <= 0.
func BigIntn(n *big.Int) *big.Int {
	r := rngpool.Get().(*RNG)
	defer rngpool.Put(r)
	return r.BigIntn(n)
}

// Float64 returns a random float64 in [0,1).
func Float64() float64 {
	r := rngpool.Get().(*RNG)
	defer rngpool.Put(r)
	return r.Float64()
}

// Perm returns a random permutation of the integers [0,n). It panics if n < 0.
func Perm(n int) []int {
	r := rngpool.Get().(*RNG)
	defer rngpool.Put(r)
	return r.Perm(n)
}

// Shuffle randomly permutes n elements by repeatedly calling swap in the range
// [0,n). It panics if n < 0.
func Shuffle(n int, swap func(i, j int)) {
	r := rngpool.Get().(*RNG)
	defer rngpool.Put(r)
	r.Shuffle(n, swap)
}

// Reader is a global, shared instance of a cryptographically strong pseudo-
// random generator. Reader is safe for concurrent use by multiple goroutines.
var Reader rngReader

type rngReader struct{}

func (rngReader) Read(b []byte) (int, error) { return Read(b) }

// A Source is a math/rand-compatible source of entropy. It is safe for
// concurrent use by multiple goroutines
type Source struct {
	rng *RNG
	mu  sync.Mutex
}

// Seed uses the provided seed value to initialize the Source to a
// deterministic state.
func (s *Source) Seed(i int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seed := make([]byte, chacha.KeySize)
	binary.LittleEndian.PutUint64(seed, uint64(i))
	for i := range s.rng.buf {
		s.rng.buf[i] = 0
	}
	chacha.XORKeyStream(s.rng.buf, s.rng.buf, make([]byte, chacha.NonceSize), seed, s.rng.rounds)
	s.rng.n = chacha.KeySize
}

// Int63 returns a non-negative random 63-bit integer as an int64.
func (s *Source) Int63() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return int64(s.rng.Uint64n(1 << 63))
}

// Uint64 returns a random 64-bit integer.
func (s *Source) Uint64() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rng.Uint64n(math.MaxUint64)
}

// NewSource returns a source for use with the math/rand package.
func NewSource() *Source { return &Source{rng: New()} }
