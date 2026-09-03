package memory

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/oklog/ulid/v2"
)

// a lease ID is a ULID whose 48 bit timestamp is the lease expiry, which every
// consumer relies on.  the 80 entropy bits carry the slab sequence number and
// the manager nonce, so a lease record is found by arithmetic and a lease
// from another Manager is rejected as unknown.
//
//	[0:6]   expiry unix ms, big endian
//	[6:12]  seq, 48 bits big endian: 4 bits shard, 44 bits counter
//	[12:16] nonce uint32, big endian
//
// with many managers in a fleet, two of them share a nonce one time in four
// billion.  a lease that reaches the wrong manager is still refused unless its
// seq names a live slot there whose expiry is the very millisecond in the ID,
// which release and extend check.

func encodeLeaseID(expiresAtMS int64, seq uint64, nonce uint32) ulid.ULID {
	var id ulid.ULID
	// SetTime only fails above the 48 bit maximum, a year far past any lease
	_ = id.SetTime(uint64(expiresAtMS))
	id[6] = byte(seq >> 40)
	id[7] = byte(seq >> 32)
	binary.BigEndian.PutUint32(id[8:12], uint32(seq))
	binary.BigEndian.PutUint32(id[12:16], nonce)
	return id
}

func decodeLeaseID(id ulid.ULID) (expiresAtMS int64, seq uint64, nonce uint32) {
	seq = uint64(id[6])<<40 | uint64(id[7])<<32 | uint64(binary.BigEndian.Uint32(id[8:12]))
	return int64(id.Time()), seq, binary.BigEndian.Uint32(id[12:16])
}

// randomNonce draws the per Manager nonce.
func randomNonce() (uint32, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b[:]), nil
}
