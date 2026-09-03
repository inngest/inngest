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

func TestTTLMapSweep(t *testing.T) {
	m := newTTLMap[int]()
	for k := uint64(0); k < 1_000; k++ {
		expiry := int64(1_000)
		if k%2 == 0 {
			expiry = 2_000
		}
		m.set(k, int(k), expiry)
	}
	require.Equal(t, 1_000, m.len())

	require.Equal(t, 0, m.sweep(999), "nothing expired yet")
	require.Equal(t, 1_000, m.len())

	require.Equal(t, 500, m.sweep(1_000))
	require.Equal(t, 500, m.len())
	for k := uint64(0); k < 1_000; k++ {
		_, ok := m.get(1_000, k)
		require.Equal(t, k%2 == 0, ok, "key %d", k)
	}

	require.Equal(t, 500, m.sweep(2_000))
	require.Equal(t, 0, m.len())
}
