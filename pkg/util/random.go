package util

import (
	"math/rand/v2"
)

func RandPerm(n int) []int {
	return rand.Perm(n)
}
