package slice

import (
	"strconv"
)

// Contains Check item in slice string type
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ContainsRune Check item in map rune type
func ContainsRune(set map[rune]struct{}, item rune) bool {
	_, ok := set[item]
	return ok
}

// ContainsValue check if value exists in slice, no matter its type
func ContainsValue(slice []interface{}, value interface{}) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// IntToString Convert slice int to slice string
func IntToString(intSl []int) (str []string) {
	for i := range intSl {
		number := intSl[i]
		text := strconv.Itoa(number)
		str = append(str, text)
	}
	return str
}
