//go:build amd64 && !appengine
// +build amd64,!appengine

package roaring

import "golang.org/x/sys/cpu"

// fillLeastSignificant16bitsVector is implemented in fillbits_vbmi2_amd64.s.
// It is only called when useVectorFill reports that the CPU can run it.
//
//go:noescape
func fillLeastSignificant16bitsVector(bitmap []uint64, x []uint32, pos int, mask uint32) int

// useVectorFill reports whether the assembly decoder may be used. It needs
// VPCOMPRESSB, that is AVX512_VBMI2 (Ice Lake and later, Zen 4 and later).
// x/sys/cpu also verifies that the operating system saves the opmask and ZMM
// state, and it honors GODEBUG=cpu.avx512vbmi2=off, so the vector path can be
// turned off at run time without rebuilding.
var useVectorFill = cpu.X86.HasAVX512VBMI2
