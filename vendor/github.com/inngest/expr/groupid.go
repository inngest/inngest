package expr

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
)

// groupID represents a group ID.  The first 2 byets are an int16 size of the expression group,
// representing the number of predicates within the expression. The last 6 bytes are a random
// ID for the predicate group.
type groupID [8]byte

var rander = rand.Read

type RandomReader func(p []byte) (n int, err error)

func (g groupID) String() string {
	return hex.EncodeToString(g[:])
}

func (g groupID) Size() uint16 {
	return binary.NativeEndian.Uint16(g[0:2])
}

func newGroupID(size uint16) groupID {
	return newGroupIDWithReader(size, rander)
}

func newGroupIDWithReader(size uint16, rander RandomReader) groupID {
	id := make([]byte, 8)
	binary.NativeEndian.PutUint16(id, size)
	_, _ = rander(id[2:])
	return [8]byte(id[0:8])
}
