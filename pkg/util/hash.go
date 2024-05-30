package util

import (
	"fmt"
	"strconv"

	"github.com/cespare/xxhash/v2"
)

func XXHash(in any) string {
	switch v := in.(type) {
	case string:
		ui := xxhash.Sum64String(v)
		return strconv.FormatUint(ui, 36)
	default:
		ui := xxhash.Sum64String(fmt.Sprintf("%v", in))
		return strconv.FormatUint(ui, 36)
	}
}
