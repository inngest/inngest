package roaring

import "math/bits"

// Portable implementations of the bitmap-container word operations. Each writes
// dst[i] = a[i] op b[i]; the Card variants also return the population count of
// the result, which callers would otherwise obtain with a second pass.
//
// dst may alias a or b: every element is read before it is written.

func orSliceGo(dst, a, b []uint64) {
	for i := range dst {
		dst[i] = a[i] | b[i]
	}
}

func andSliceGo(dst, a, b []uint64) {
	for i := range dst {
		dst[i] = a[i] & b[i]
	}
}

func xorSliceGo(dst, a, b []uint64) {
	for i := range dst {
		dst[i] = a[i] ^ b[i]
	}
}

func andNotSliceGo(dst, a, b []uint64) {
	for i := range dst {
		dst[i] = a[i] &^ b[i]
	}
}

func orCardSliceGo(dst, a, b []uint64) uint64 {
	card := 0
	for i := range dst {
		v := a[i] | b[i]
		dst[i] = v
		card += bits.OnesCount64(v)
	}
	return uint64(card)
}

func andCardSliceGo(dst, a, b []uint64) uint64 {
	card := 0
	for i := range dst {
		v := a[i] & b[i]
		dst[i] = v
		card += bits.OnesCount64(v)
	}
	return uint64(card)
}
