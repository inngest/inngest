package expr

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
)

// groupID represents a group ID.  Layout, in bytes:
// - 2: an int16 size of the expression group,
// - 1: optimization flag, for optimizing "!=" in string matching
// - 5: random ID for group
type groupID [8]byte

// type internedGroupID unique.Handle[groupID]
//
// func (i internedGroupID) Value() groupID {
// 	return unique.Handle[groupID](i).Value()
// }
//
// func (i internedGroupID) Size() uint16 {
// 	// Uses unsafe pointers to access the underlying groupID
// 	// to return the size without a copy.
// 	handlePtr := unsafe.Pointer(&i)
// 	unsafe.Slice(
// 	// return (*groupID)(unsafe.Pointer(unsafe.SliceData(([8]byte)(handlePtr)))).Size()
// }

var rander = rand.Read

type RandomReader func(p []byte) (n int, err error)

const (
	OptimizeNone = 0x0
)

func (g groupID) String() string {
	return hex.EncodeToString(g[:])
}

func (g groupID) Size() uint16 {
	return binary.NativeEndian.Uint16(g[0:2])
}

func (g groupID) Flag() byte {
	return g[2]
}

func newGroupID(size uint16, optimizeFlag byte) groupID {
	return newGroupIDWithReader(size, optimizeFlag, rander)
}

func newGroupIDWithReader(size uint16, optimizeFlag byte, rander RandomReader) groupID {
	id := make([]byte, 8)
	binary.NativeEndian.PutUint16(id, size)
	// Set the optimize byte.
	id[2] = optimizeFlag
	_, _ = rander(id[3:])

	gid := groupID([8]byte(id[0:8]))
	// interned := internedGroupID(unique.Make(gid))
	return gid
}
