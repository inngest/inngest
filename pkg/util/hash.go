package util

import (
	"fmt"
	"strconv"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
)

func XXHash(in any) string {
	switch v := in.(type) {
	case string:
		ui := xxhash.Sum64String(v)
		return strconv.FormatUint(ui, 36)
	case uuid.UUID:
		ui := xxhash.Sum64(v[:])
		return strconv.FormatUint(ui, 36)
	case []byte:
		ui := xxhash.Sum64(v)
		return strconv.FormatUint(ui, 36)
	default:
		ui := xxhash.Sum64String(fmt.Sprintf("%v", in))
		return strconv.FormatUint(ui, 36)
	}
}

func XXHashFloat(in any) float64 {
	switch v := in.(type) {
	case string:
		ui := xxhash.Sum64String(v)
		return float64(ui)
	case uuid.UUID:
		ui := xxhash.Sum64(v[:])
		return float64(ui)
	case []byte:
		ui := xxhash.Sum64(v)
		return float64(ui)
	default:
		ui := xxhash.Sum64String(fmt.Sprintf("%v", in))
		return float64(ui)
	}
}
