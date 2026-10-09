package quack

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuackWireUnsignedLeb128RoundTrip(t *testing.T) {
	for _, v := range []uint64{0, 1, 127, 128, 300, 1 << 32, ^uint64(0)} {
		w := &writer{}
		w.writeUnsignedLeb128(v)

		r := newReader(w.bytes())
		got, err := r.readUnsignedLeb128()
		require.NoError(t, err)
		require.Equal(t, v, got)
	}
}

func TestQuackWireSignedLeb128RoundTrip(t *testing.T) {
	for _, v := range []int64{0, 1, -1, 63, -64, 64, -65, 1 << 40, -(1 << 40)} {
		w := &writer{}
		w.writeSignedLeb128(v)

		r := newReader(w.bytes())
		got, err := r.readSignedLeb128()
		require.NoError(t, err)
		require.Equal(t, v, got)
	}
}

func TestQuackWireObjectTerminatorRoundTrip(t *testing.T) {
	w := &writer{}
	w.beginObject()
	w.writeUint64(1, 42)
	w.endObject()

	r := newReader(w.bytes())
	r.beginObject()
	require.NoError(t, r.beginProperty(1))
	v, err := r.readUInt64()
	require.NoError(t, err)
	require.Equal(t, uint64(42), v)
	require.NoError(t, r.endObject())
}

func TestQuackWireEndObjectErrorsOnMissingTerminator(t *testing.T) {
	w := &writer{}
	w.writeUint64(1, 42) // no endObject() -> no terminator written

	r := newReader(w.bytes())
	_, err := r.readUInt64() // consume the raw LEB128 value directly
	require.NoError(t, err)
	err = r.endObject()
	require.Error(t, err)
}

func TestQuackWireTryBeginPropertyOmittedField(t *testing.T) {
	w := &writer{}
	w.beginObject()
	w.writeStringDefault(2, "") // default-omitted: field 2 never written
	w.writeUint64(3, 7)
	w.endObject()

	r := newReader(w.bytes())
	r.beginObject()
	present, err := r.tryBeginProperty(2)
	require.NoError(t, err)
	require.False(t, present)

	require.NoError(t, r.beginProperty(3))
	v, err := r.readUInt64()
	require.NoError(t, err)
	require.Equal(t, uint64(7), v)
	require.NoError(t, r.endObject())
}

func TestQuackWireStringRoundTrip(t *testing.T) {
	w := &writer{}
	w.writeString(1, "hello, 世界")

	r := newReader(w.bytes())
	require.NoError(t, r.beginProperty(1))
	got, err := r.readString()
	require.NoError(t, err)
	require.Equal(t, "hello, 世界", got)
}

func TestQuackWireBoolRoundTrip(t *testing.T) {
	w := &writer{}
	w.writeBool(1, true)

	r := newReader(w.bytes())
	require.NoError(t, r.beginProperty(1))
	got, err := r.readBool()
	require.NoError(t, err)
	require.True(t, got)
}

func TestQuackWireDataMemoryRoundTrip(t *testing.T) {
	w := &writer{}
	w.writeData([]byte{0x01, 0x02, 0x03})

	r := newReader(w.bytes())
	got, err := r.readData()
	require.NoError(t, err)
	require.Equal(t, []byte{0x01, 0x02, 0x03}, got)
}

func TestQuackWireListCountRoundTrip(t *testing.T) {
	w := &writer{}
	w.beginList(3)
	w.writeUint64(1, 1)
	w.writeUint64(1, 2)
	w.writeUint64(1, 3)

	r := newReader(w.bytes())
	n, err := r.beginList()
	require.NoError(t, err)
	require.EqualValues(t, 3, n)
	for i := 0; i < 3; i++ {
		require.NoError(t, r.beginProperty(1))
		_, err := r.readUInt64()
		require.NoError(t, err)
	}
}

func TestQuackWireReadUnsignedLeb128TruncatedErrors(t *testing.T) {
	// 0x80 alone signals "more bytes follow" but none do.
	r := newReader([]byte{0x80})
	_, err := r.readUnsignedLeb128()
	require.Error(t, err)
}
