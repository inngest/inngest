//go:build amd64 && !appengine
// +build amd64,!appengine

package roaring

import "golang.org/x/sys/cpu"

// The functions below are implemented in popcnt_avx512_amd64.s using
// AVX512_VPOPCNTDQ. They are used in preference to the AVX2 kernels when the
// CPU provides VPOPCNTQ (see useAVX512Popcnt).

//go:noescape
func popcntSliceAVX512(s []uint64) uint64

//go:noescape
func popcntMaskSliceAVX512(s, m []uint64) uint64

//go:noescape
func popcntAndSliceAVX512(s, m []uint64) uint64

//go:noescape
func popcntOrSliceAVX512(s, m []uint64) uint64

//go:noescape
func popcntXorSliceAVX512(s, m []uint64) uint64

// useAVX512Popcnt selects the AVX-512 implementations when the running CPU has
// AVX512_VPOPCNTDQ. x/sys/cpu verifies that the operating system saves the
// opmask and ZMM state, and honors GODEBUG=cpu.avx512vpopcntdq=off so the path
// can be disabled at run time. Evaluated once at package initialization.
var useAVX512Popcnt = cpu.X86.HasAVX512VPOPCNTDQ
