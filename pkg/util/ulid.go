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

// DeterministicAbortedDeferSpanSeed returns the seed for the SECOND
// executor.defer span emitted when a defer is aborted. It carries the
// terminal defer.status = aborted attribute.
//
// It uses a distinct "a" tag (vs. the "s" tag of the original schedule span)
// so the two spans get different dynamic span IDs and survive as separate rows
// in GetSpansByRunIDsAndName, which filters by span name and so never sees the
// EXTEND fragments that UpdateSpan would otherwise produce. GetRunDefers
// collapses the two rows by hashed ID, preferring the terminal status.
//
// Like the "s"/"c"/untagged siblings above, determinism keeps the span ID
// stable across abort retries so resolvers don't see duplicate abort spans.
func DeterministicAbortedDeferSpanSeed(parent ulid.ULID, hashedID string) []byte {
	return []byte(parent.String() + hashedID + "a")
}

// DeterministicChildRunIDDeferSpanSeed returns the seed for the THIRD
// executor.defer span emitted when a deferred child run is scheduled. It carries
// the defer.child_run_id attribute linking the parent's defer to the actual
// child run ID.
//
// The child run ID isn't known when the schedule span is emitted (the child
// hasn't been scheduled yet), so it's recorded on a separate span at
// child-schedule time. We emit a distinct span rather than extending the
// schedule span because the linkage query GetSpansByRunIDsAndName filters by
// name = "executor.defer" and never reads the EXTEND fragments UpdateSpan
// produces — the same reason the abort span exists. GetRunDefers collapses the
// rows by hashed ID, carrying the child run ID onto the surfaced defer.
//
// It uses a distinct "c" tag (vs. "s" schedule / "a" abort) so it gets its own
// dynamic span ID and survives as its own row. Determinism keeps the span ID
// stable across schedule retries so resolvers don't see duplicates.
func DeterministicChildRunIDDeferSpanSeed(parent ulid.ULID, hashedID string) []byte {
	return []byte(parent.String() + hashedID + "c")
}
