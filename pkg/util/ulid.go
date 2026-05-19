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
// its parent run ID and the defer's hashed ID. Determinism lets callers
// idempotently look up the child run from the parent context.
//
// "r"-tagged sibling of DeterministicDeferEventID (untagged) and
// DeterministicDeferSpanSeed ("s"); the trio shares the same
// (parent, hashedID) input and differs only by the trailing tag byte.
// Changing a tag requires updating all three in lockstep.
func DeterministicChildRunID(parent ulid.ULID, hashedID string) ulid.ULID {
	// err is unreachable: ulid.New only fails when its entropy reader errors,
	// and bytes.Reader over a fixed SHA-256 sum cannot.
	id, _ := DeterministicULID(ulid.Time(parent.Time()), []byte(parent.String()+hashedID+"r"))
	return id
}

// DeterministicDeferEventID returns the inngest/deferred.schedule event ID
// for a single defer. Determinism dedupes duplicate publishes on the runner
// side, which uses event.ID as the schedule idempotency key.
func DeterministicDeferEventID(parent ulid.ULID, hashedID string) ulid.ULID {
	id, _ := DeterministicULID(ulid.Time(parent.Time()), []byte(parent.String()+hashedID))
	return id
}

// DeterministicDeferSpanSeed returns the seed for the executor.defer span.
// Determinism keeps the span ID stable across finalize retries so resolvers
// don't see duplicates.
func DeterministicDeferSpanSeed(parent ulid.ULID, hashedID string) []byte {
	return []byte(parent.String() + hashedID + "s")
}
