package memory

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestLeaseIDRoundTrip(t *testing.T) {
	expiry := time.Date(2026, 3, 4, 5, 6, 7, 890_000_000, time.UTC)
	expiryMS := expiry.UnixMilli()

	id := encodeLeaseID(expiryMS, 42, 0xBEEF)

	gotExpiry, gotSeq, gotNonce := decodeLeaseID(id)
	require.Equal(t, expiryMS, gotExpiry)
	require.Equal(t, uint64(42), gotSeq)
	require.Equal(t, uint16(0xBEEF), gotNonce)

	// consumers read the expiry from the ULID timestamp
	require.Equal(t, uint64(expiryMS), id.Time())
	require.Equal(t, expiry.Truncate(time.Millisecond), ulid.Time(id.Time()).UTC())

	parsed, err := ulid.Parse(id.String())
	require.NoError(t, err)
	require.Equal(t, id, parsed)
}

func TestLeaseIDDistinguishesSeqAndNonce(t *testing.T) {
	a := encodeLeaseID(1_000, 1, 1)
	b := encodeLeaseID(1_000, 2, 1)
	c := encodeLeaseID(1_000, 1, 2)
	require.NotEqual(t, a, b)
	require.NotEqual(t, a, c)

	_, _, nonce := decodeLeaseID(c)
	require.Equal(t, uint16(2), nonce)

	// the ULID string order follows the seq inside one millisecond
	require.Less(t, a.String(), b.String())

	big := encodeLeaseID(1<<48-1, 1<<64-1, 1<<16-1)
	e, s, n := decodeLeaseID(big)
	require.Equal(t, int64(1<<48-1), e)
	require.Equal(t, uint64(1<<64-1), s)
	require.Equal(t, uint16(1<<16-1), n)
}

func TestLeaseIDZeroIsUnknown(t *testing.T) {
	e, s, n := decodeLeaseID(ulid.Zero)
	require.Zero(t, e)
	require.Zero(t, s, "seq 0 is never allocated, so a zero ULID never matches a lease")
	require.Zero(t, n)
}

func TestRandomNonce(t *testing.T) {
	_, err := randomNonce()
	require.NoError(t, err)
}
