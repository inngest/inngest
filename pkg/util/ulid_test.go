package util

import (
	"bytes"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestDeterministicULID(t *testing.T) {
	ts := time.UnixMilli(1_700_000_000_000) // 2023-11-14

	t.Run("same inputs produce the same ULID", func(t *testing.T) {
		a, err := DeterministicULID(ts, []byte("seed"))
		require.NoError(t, err)
		b, err := DeterministicULID(ts, []byte("seed"))
		require.NoError(t, err)
		require.Equal(t, a, b)
	})

	t.Run("different seeds produce different ULIDs", func(t *testing.T) {
		a, err := DeterministicULID(ts, []byte("seed-a"))
		require.NoError(t, err)
		b, err := DeterministicULID(ts, []byte("seed-b"))
		require.NoError(t, err)
		require.NotEqual(t, a, b)
	})

	t.Run("different timestamps produce different ULIDs for the same seed", func(t *testing.T) {
		a, err := DeterministicULID(ts, []byte("seed"))
		require.NoError(t, err)
		b, err := DeterministicULID(ts.Add(time.Second), []byte("seed"))
		require.NoError(t, err)
		require.NotEqual(t, a, b,
			"time prefix must vary with timestamp, otherwise the function isn't building a real ULID")
	})

	t.Run("time prefix encodes the timestamp", func(t *testing.T) {
		id, err := DeterministicULID(ts, []byte("seed"))
		require.NoError(t, err)
		require.Equal(t, uint64(ts.UnixMilli()), id.Time(),
			"ts must round-trip through the ULID's first 6 bytes; if this fails the helper isn't producing a valid ULID")
	})

	t.Run("output round-trips through ulid.Parse", func(t *testing.T) {
		id, err := DeterministicULID(ts, []byte("seed"))
		require.NoError(t, err)
		parsed, err := ulid.Parse(id.String())
		require.NoError(t, err)
		require.Equal(t, id, parsed)
	})

	t.Run("nil seed does not panic and produces a valid ULID", func(t *testing.T) {
		id, err := DeterministicULID(ts, nil)
		require.NoError(t, err)
		parsed, err := ulid.Parse(id.String())
		require.NoError(t, err)
		require.Equal(t, id, parsed)
	})

	t.Run("seed shorter than 10 bytes is zero-padded", func(t *testing.T) {
		// The helper's documented behavior: short seeds get zero-padded to 10
		// bytes before hashing. So "hi" (2 bytes) and its zero-padded form
		// (10 bytes ending in 8 zero bytes) must produce the same ULID.
		short, err := DeterministicULID(ts, []byte("hi"))
		require.NoError(t, err)
		padded := make([]byte, 10)
		copy(padded, "hi")
		full, err := DeterministicULID(ts, padded)
		require.NoError(t, err)
		require.Equal(t, short, full,
			"short seed and its zero-padded-to-10-bytes form must be equivalent")
	})

	t.Run("long seed hashes deterministically", func(t *testing.T) {
		// Longer than the SHA-256 input size we'd ever hit in practice; the
		// hash must still produce a stable result on repeat invocations.
		long := bytes.Repeat([]byte("x"), 1024)
		a, err := DeterministicULID(ts, long)
		require.NoError(t, err)
		b, err := DeterministicULID(ts, long)
		require.NoError(t, err)
		require.Equal(t, a, b)
	})
}

func TestDeterministicDeferEventID(t *testing.T) {
	fixedParent := ulid.MustParse("01HKQJZ5R7XR4MNTQGZ8Z3KPAB")
	require.Equal(t,
		"01HKQJZ5R7CYQKSKMYBEBKYTBM",
		DeterministicDeferEventID(fixedParent, "fixed-hashed-id").String())
}

func TestDeterministicDeferSpanSeed(t *testing.T) {
	fixedParent := ulid.MustParse("01HKQJZ5R7XR4MNTQGZ8Z3KPAB")

	// The defer seeds share the (parent, hashedID) input and differ only by
	// the trailing tag. If the schedule span seed ever collided with the
	// "c"-tag child-run-id span seed, the "a"-tag abort span seed, or the
	// untagged event ID seed, the deterministic span IDs would clash.
	span := DeterministicDeferSpanSeed(fixedParent, "fixed-hashed-id")
	require.NotEqual(t, DeterministicChildRunIDDeferSpanSeed(fixedParent, "fixed-hashed-id"), span)
	require.NotEqual(t, DeterministicAbortedDeferSpanSeed(fixedParent, "fixed-hashed-id"), span)
	require.NotEqual(t, []byte(fixedParent.String()+"fixed-hashed-id"), span)
}

func TestDeterministicAbortedDeferSpanSeed(t *testing.T) {
	fixedParent := ulid.MustParse("01HKQJZ5R7XR4MNTQGZ8Z3KPAB")

	// The abort span MUST get a different dynamic span ID than the schedule
	// span so both survive as separate rows in the linkage query. If the
	// "a" and "s" seeds ever collided, the abort span would overwrite the
	// schedule span instead of being collapsed alongside it.
	require.NotEqual(t,
		DeterministicDeferSpanSeed(fixedParent, "fixed-hashed-id"),
		DeterministicAbortedDeferSpanSeed(fixedParent, "fixed-hashed-id"))
}

func TestDeterministicChildRunIDDeferSpanSeed(t *testing.T) {
	fixedParent := ulid.MustParse("01HKQJZ5R7XR4MNTQGZ8Z3KPAB")

	// The child-run-id span MUST get a different dynamic span ID than the
	// schedule and abort spans so all three survive as separate rows in the
	// linkage query, where GetRunDefers collapses them by hashed ID.
	child := DeterministicChildRunIDDeferSpanSeed(fixedParent, "fixed-hashed-id")
	require.NotEqual(t, DeterministicDeferSpanSeed(fixedParent, "fixed-hashed-id"), child)
	require.NotEqual(t, DeterministicAbortedDeferSpanSeed(fixedParent, "fixed-hashed-id"), child)
}
