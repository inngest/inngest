package util

import (
	"bytes"
	"crypto/sha256"
	"time"

	"github.com/oklog/ulid/v2"
)

// DeterministicULID creates a deterministic ULID given the seed.
// new SeededID.
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

// DeterministicChildRunID derives the run ID of a deferred child run from
// its parent run ID and the defer's hashed ID. It is the "r"-tagged sibling
// of two other deterministic IDs that share the same (parent_run_id,
// hashed_id) input pair, all built in executor.buildDeferEvents:
//
//   - schedule event ID (untagged)
//   - executor.defer span ID ("s" tag)
//   - child run ID ("r" tag, this function)
//
// The tag is what differentiates these three ULIDs; do not change it
// without updating all three sites in lockstep.
func DeterministicChildRunID(parent ulid.ULID, hashedID string) ulid.ULID {
	// err is unreachable: ulid.New only fails when its entropy reader errors,
	// and bytes.Reader over a fixed SHA-256 sum cannot.
	id, _ := DeterministicULID(ulid.Time(parent.Time()), []byte(parent.String()+hashedID+"r"))
	return id
}
