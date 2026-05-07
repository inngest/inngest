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
