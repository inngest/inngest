package util

import (
	"bytes"
	"crypto/sha256"
	"time"

	"github.com/oklog/ulid/v2"
)

// DeterministicULID creates a deterministic ULID given the seed.
func DeterministicULID(ts time.Time, seed []byte) (ulid.ULID, error) {
	// ULID requires 10 bytes of entropy. If seed is shorter, pad it with zeros.
	// If seed is longer, use all of it as the entropy source.
	if len(seed) < 10 {
		entropy := make([]byte, 10)
		copy(entropy, seed)
		seed = entropy
	}
	// Hash the seed in case it's more than 10 bytes. ULID generation silently ignores bytes after the 10th
	data := sha256.Sum256(seed)
	return ulid.New(
		uint64(ts.UnixMilli()),
		bytes.NewReader(data[:]),
	)
}
