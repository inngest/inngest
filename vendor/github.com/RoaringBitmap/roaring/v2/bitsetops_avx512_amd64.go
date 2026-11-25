//go:build amd64 && !appengine
// +build amd64,!appengine

package roaring

import "golang.org/x/sys/cpu"

// The functions below are implemented in bitsetops_avx512_amd64.s. The Card
// variants fuse the write with the population count of the result, so a
// bitmap-container operation that needs both makes one pass over the container
// instead of two.

//go:noescape
func orSliceAVX512(dst, a, b []uint64)

//go:noescape
func andSliceAVX512(dst, a, b []uint64)

//go:noescape
func xorSliceAVX512(dst, a, b []uint64)

//go:noescape
func andNotSliceAVX512(dst, a, b []uint64)

//go:noescape
func orCardSliceAVX512(dst, a, b []uint64) uint64

//go:noescape
func andCardSliceAVX512(dst, a, b []uint64) uint64

// useAVX512BitsetOps requires AVX512_VPOPCNTDQ because the fused kernels use
// VPOPCNTQ; the plain writes only need AVX512F, but they are gated together so
// a single flag governs the whole file. x/sys/cpu verifies operating-system
// support for the ZMM state and honors GODEBUG=cpu.avx512vpopcntdq=off.
var useAVX512BitsetOps = cpu.X86.HasAVX512VPOPCNTDQ

func orSlice(dst, a, b []uint64) {
	if useAVX512BitsetOps {
		orSliceAVX512(dst, a, b)
		return
	}
	orSliceGo(dst, a, b)
}

func andSlice(dst, a, b []uint64) {
	if useAVX512BitsetOps {
		andSliceAVX512(dst, a, b)
		return
	}
	andSliceGo(dst, a, b)
}

func xorSlice(dst, a, b []uint64) {
	if useAVX512BitsetOps {
		xorSliceAVX512(dst, a, b)
		return
	}
	xorSliceGo(dst, a, b)
}

func andNotSlice(dst, a, b []uint64) {
	if useAVX512BitsetOps {
		andNotSliceAVX512(dst, a, b)
		return
	}
	andNotSliceGo(dst, a, b)
}

func orCardSlice(dst, a, b []uint64) uint64 {
	if useAVX512BitsetOps {
		return orCardSliceAVX512(dst, a, b)
	}
	// Without a vector population count the fused loop is no faster than the
	// two passes the callers used before, and popcntSlice may itself be AVX2.
	orSliceGo(dst, a, b)
	return popcntSlice(dst)
}

func andCardSlice(dst, a, b []uint64) uint64 {
	if useAVX512BitsetOps {
		return andCardSliceAVX512(dst, a, b)
	}
	andSliceGo(dst, a, b)
	return popcntSlice(dst)
}
