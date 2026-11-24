package memory

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTTLMapExpiryBoundary(t *testing.T) {
	m := newTTLMap[string]()
	m.set(7, "a", 1_000)

	v, ok := m.get(999, 7)
	require.True(t, ok)
	require.Equal(t, "a", v)

	_, ok = m.get(1_000, 7)
	require.False(t, ok, "present while now < expiry, like Redis EX")

	_, ok = m.get(0, 8)
	require.False(t, ok)

	m.set(7, "b", 5_000)
	v, ok = m.get(1_000, 7)
	require.True(t, ok)
	require.Equal(t, "b", v, "set replaces the value and the expiry")
	require.Equal(t, 1, m.len())
}

// TestTTLMapRotate checks that rotate frees an entry only after it expired,
// and that get finds an entry in the old map until then.
func TestTTLMapRotate(t *testing.T) {
	m := newTTLMap[int]()
	for k := range uint64(1_000) {
		m.set(k, int(k), 1_000)
	}

	m.rotate(999)
	require.Equal(t, 1_000, m.len(), "nothing expired yet")
	for k := range uint64(1_000) {
		v, ok := m.get(999, k)
		require.True(t, ok, "key %d", k)
		require.Equal(t, int(k), v)
	}

	for k := uint64(1_000); k < 2_000; k++ {
		m.set(k, int(k), 2_000)
	}
	m.rotate(999)
	require.Equal(t, 2_000, m.len(), "the old map has entries that did not expire")

	m.rotate(1_000)
	require.Equal(t, 1_000, m.len())
	for k := range uint64(2_000) {
		_, ok := m.get(1_000, k)
		require.Equal(t, k >= 1_000, ok, "key %d", k)
	}

	m.rotate(2_000)
	require.Equal(t, 0, m.len())
	require.Zero(t, m.lastExpiry())
}

// TestTTLMapRotateAllExpired checks that one rotate frees a map in which
// every entry expired.
func TestTTLMapRotateAllExpired(t *testing.T) {
	m := newTTLMap[int]()
	for k := range uint64(1_000) {
		m.set(k, int(k), 1_000)
	}
	m.rotate(1_000)
	require.Equal(t, 0, m.len())
}

// TestTTLMapSetAfterRotate checks that a key set again after a rotate has
// the new value and the new expiry, although the old map still has the key.
func TestTTLMapSetAfterRotate(t *testing.T) {
	m := newTTLMap[string]()
	m.set(7, "a", 1_000)
	m.rotate(0)
	m.set(7, "b", 500)
	require.Equal(t, 2, m.len())
	require.Equal(t, int64(1_000), m.lastExpiry())

	v, ok := m.get(499, 7)
	require.True(t, ok)
	require.Equal(t, "b", v)
	_, ok = m.get(500, 7)
	require.False(t, ok, "the first value does not come back when the second expires")

	var got []string
	m.each(0, func(_ uint64, v string, expiresAtMS int64) {
		require.Equal(t, int64(500), expiresAtMS)
		got = append(got, v)
	})
	require.Equal(t, []string{"b"}, got)
}

// TestTTLMapRotateKeepsLongEntry checks that a stripe keeps its old map
// while one entry in it has not expired.
func TestTTLMapRotateKeepsLongEntry(t *testing.T) {
	m := newTTLMap[int]()
	// the three keys are in one stripe
	long, short, later := uint64(1), uint64(1+ttlStripes), uint64(1+2*ttlStripes)
	m.set(long, 1, 10_000)
	m.set(short, 2, 1_000)
	m.rotate(0)
	m.set(later, 3, 1_500)

	m.rotate(5_000)
	require.Equal(t, 3, m.len())
	_, ok := m.get(5_000, long)
	require.True(t, ok)
	_, ok = m.get(5_000, short)
	require.False(t, ok)
	_, ok = m.get(5_000, later)
	require.False(t, ok)

	m.rotate(10_000)
	require.Equal(t, 0, m.len())
}
