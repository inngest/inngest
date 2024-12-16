//go:build !go1.22
// +build !go1.22

package util

import (
	"encoding/binary"
	"math/rand"
	"sync"
	"time"
)

var sources sync.Pool

func init() {
	sources = sync.Pool{New: func() any { return rand.New(rand.NewSource(time.Now().UnixNano())) }}
}

func Shuffle(n int, swap func(i, j int)) {
	rand.Shuffle(n, swap)
}

var rngPool = sync.Pool{
	New: func() any {
		return rand.New(rand.NewSource(time.Now().UnixNano()))
	},
}

func FastRand(n int) (r int) {
	s := rngPool.Get().(*rand.Rand)
	r = s.Intn(n)
	rngPool.Put(s)
	return
}

func RandomBytes() []byte {
	val := make([]byte, 24)
	src := sources.Get().(rand.Source64)
	binary.BigEndian.PutUint64(val[0:8], src.Uint64())
	binary.BigEndian.PutUint64(val[8:16], src.Uint64())
	binary.BigEndian.PutUint64(val[16:24], src.Uint64())
	sources.Put(src)
	return val
}
