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

	id := encodeLeaseID(expiryMS, 42, 0xDEADBEEF)

	gotExpiry, gotSeq, gotNonce := decodeLeaseID(id)
	require.Equal(t, expiryMS, gotExpiry)
	require.Equal(t, uint64(42), gotSeq)
	require.Equal(t, uint32(0xDEADBEEF), gotNonce)

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
	require.Equal(t, uint32(2), nonce)

	// the ULID string order follows the seq inside one millisecond
	require.Less(t, a.String(), b.String())

	big := encodeLeaseID(1<<48-1, 1<<48-1, 1<<32-1)
	e, s, n := decodeLeaseID(big)
	require.Equal(t, int64(1<<48-1), e)
	require.Equal(t, uint64(1<<48-1), s)
	require.Equal(t, uint32(1<<32-1), n)

	// every shard's seqs fit in 48 bits
	top := uint64(slabShards-1)<<shardShift | 1<<shardShift - 1
	require.Less(t, top, uint64(1)<<48)
	_, s, _ = decodeLeaseID(encodeLeaseID(1_000, top, 0))
	require.Equal(t, top, s)
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
