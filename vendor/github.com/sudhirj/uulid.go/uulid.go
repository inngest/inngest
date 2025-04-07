package uulid

import (
	"bytes"
	cryptoRand "crypto/rand"
	"crypto/sha1"
	"hash"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// UULID represents a 16 byte, or a 128-bit number, which is exactly the same representation as
// UUIDs and ULIDs. This allows for easy movement between each representation.
type UULID [16]byte

// AsUUID returns the UULID as a UUID, which will represent it itself in the format `XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX`
func (uulid UULID) AsUUID() uuid.UUID {
	return uuid.UUID(uulid)
}

// AsULID returns the UULID as a ULID, which will represent it itself as a 26 character Base32 string, an example being `01ARZ3NDEKTSV4RRFFQ69G5FAV`
func (uulid UULID) AsULID() ulid.ULID {
	return ulid.ULID(uulid)
}

// ULIDString returns the Base32 ULID representation, which occupies 26 characters, like `01ARZ3NDEKTSV4RRFFQ69G5FAV`
func (uulid UULID) ULIDString() string {
	return uulid.AsULID().String()
}

// UUIDString returns the hex encoded UUID format that looks like `XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX`
func (uulid UULID) UUIDString() string {
	return uulid.AsUUID().String()
}

// String returns the default of a UULID, which is the ULID representation.
func (uulid UULID) String() string {
	return uulid.ULIDString()
}

func MustParseULID(s string) UULID {
	return FromULID(ulid.MustParse(s))
}

func MustParseUUID(s string) UULID {
	return FromUUID(uuid.MustParse(s))
}

// ParseULID parses ULID, returning an error in case of failure.
func ParseULID(s string) (UULID, error) {
	ulid, err := ulid.Parse(s)
	if err != nil {
		return UULID{}, err
	}
	return FromULID(ulid), nil
}

// ParseUUID parses UUID, returning an error in case of failure.
func ParseUUID(s string) (UULID, error) {
	uuid, err := uuid.Parse(s)
	if err != nil {
		return UULID{}, err
	}
	return FromUUID(uuid), nil
}

func FromUUID(uuid uuid.UUID) UULID {
	return UULID(uuid)
}

func FromULID(ulid ulid.ULID) UULID {
	return UULID(ulid)
}

// NewTimeOnlyUULID returns a purely time based ID with no random component (all zeroes).
// This allows using it to query for IDs after or before a given time.
//
// The ULID representation looks like `01DW6SF6P70000000000000000`, which allows for
// storage and querying in any datastore, even if byte arrays are unsupported.
//
// The UUID representation looks like `016f0d97-9ac7-0000-0000-000000000000`, which allows for
// range based queries even if UUID is internally stored as a byte array (common in Postgres, etc).
func NewTimeOnlyUULID(t time.Time) UULID {
	return FromULID(ulid.MustNew(ulid.Timestamp(t), zeroReader{}))
}

// NewContentUULID returns a UULID with the given time component and the "randomness" component
// filled with bytes from the SHA1 hash of the bytes in the given reader. Note that this ID is no
// longer random - generation of the ID is now completely functional and idempotent. The combination
// of the given timestamp and content will always generate the same ID.
//
// This is especially useful for assigning IDs to immutable pieces of chronological data, where the
// meaning of the data is clearly defined by one or more of its attributes.
//
// Note that the ULID requires 10 bytes to complete the ID after the timestamp - but because the data
// is being SHA1 hashed, you do not need to provide 10 bytes in the reader. Any number of bytes will do.
// Also note that only half the SHA1 digest is being used in the ID (SHA1 gives 20 bytes, of which we use 10),
// so that needs to be taken into account in determining any collision rates.
func NewContentUULID(timestamp time.Time, reader io.Reader) UULID {
	return NewContentHashedUULID(timestamp, reader, sha1.New())
}

func NewContentHashedUULID(timestamp time.Time, reader io.Reader, hasher hash.Hash) UULID {
	_, _ = io.Copy(hasher, reader)
	digest := hasher.Sum(nil)
	return FromULID(ulid.MustNew(ulid.Timestamp(timestamp), bytes.NewReader(digest[:])))
}

func NowUULID() UULID {
	return UULID(ulid.MustNew(ulid.Now(), cryptoRand.Reader))
}

func NewTimedUULID(t time.Time) UULID {
	return UULID(ulid.MustNew(ulid.Timestamp(t), cryptoRand.Reader))
}

type zeroReader struct{}

func (zr zeroReader) Read(b []byte) (int, error) {
	b[0] = 0
	return 1, nil
}
