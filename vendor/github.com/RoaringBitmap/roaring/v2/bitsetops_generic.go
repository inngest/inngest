//go:build !amd64 || appengine
// +build !amd64 appengine

package roaring

func orSlice(dst, a, b []uint64)     { orSliceGo(dst, a, b) }
func andSlice(dst, a, b []uint64)    { andSliceGo(dst, a, b) }
func xorSlice(dst, a, b []uint64)    { xorSliceGo(dst, a, b) }
func andNotSlice(dst, a, b []uint64) { andNotSliceGo(dst, a, b) }

func orCardSlice(dst, a, b []uint64) uint64  { return orCardSliceGo(dst, a, b) }
func andCardSlice(dst, a, b []uint64) uint64 { return andCardSliceGo(dst, a, b) }
