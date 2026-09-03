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
//	[6:14]  seq uint64, big endian
//	[14:16] nonce uint16, big endian

func encodeLeaseID(expiresAtMS int64, seq uint64, nonce uint16) ulid.ULID {
	var id ulid.ULID
	// SetTime only fails above the 48 bit maximum, a year far past any lease
	_ = id.SetTime(uint64(expiresAtMS))
	binary.BigEndian.PutUint64(id[6:14], seq)
	binary.BigEndian.PutUint16(id[14:16], nonce)
	return id
}

func decodeLeaseID(id ulid.ULID) (expiresAtMS int64, seq uint64, nonce uint16) {
	return int64(id.Time()), binary.BigEndian.Uint64(id[6:14]), binary.BigEndian.Uint16(id[14:16])
}

// randomNonce draws the per Manager nonce.
func randomNonce() (uint16, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b[:]), nil
}
