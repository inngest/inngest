//go:build !amd64 || appengine
// +build !amd64 appengine

package roaring

const useVectorFill = false

func fillLeastSignificant16bitsVector(bitmap []uint64, x []uint32, pos int, mask uint32) int {
	return fillLeastSignificant16bitsScalar(bitmap, x, pos, mask)
}
