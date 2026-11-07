package avro

import (
	"errors"
	"math/bits"
)

func appendUint32(dst []byte, u uint32) []byte {
	return append(dst, byte(u), byte(u>>8), byte(u>>16), byte(u>>24))
}

func appendUint64(dst []byte, u uint64) []byte {
	return append(dst, byte(u), byte(u>>8), byte(u>>16), byte(u>>24),
		byte(u>>32), byte(u>>40), byte(u>>48), byte(u>>56))
}

// uvarintLens maps bits.Len32's result to the uvarint byte length. Only
// indices 0..32 are ever read — that is bits.Len32's whole range — but the
// table is 256 long so the compiler can prove byte(bits.Len32(u)) is in
// bounds and drop the check.
//
// The entries past index 32 are unreachable padding and are NOT a valid
// 64-bit extension: they were written in decimal notation, so index 64
// holds \x10 (16), not the 10 a uvarlong would need. uvarintLen is the
// 32-bit lookup and nothing else; appendVarlong does its own magnitude
// switch and never reads this.
const uvarintLens = "\x01\x01\x01\x01\x01\x01\x01\x01\x02\x02\x02\x02\x02\x02\x02\x03\x03\x03\x03\x03\x03\x03\x04\x04\x04\x04\x04\x04\x04\x05\x05\x05\x05\x05\x05\x05\x06\x06\x06\x06\x06\x06\x06\x07\x07\x07\x07\x07\x07\x07\x08\x08\x08\x08\x08\x08\x08\x09\x09\x09\x09\x09\x09\x09\x10\x10\x10\x10\x10\x10\x10\x11\x11\x11\x11\x11\x11\x11\x12\x12\x12\x12\x12\x12\x12\x13\x13\x13\x13\x13\x13\x13\x14\x14\x14\x14\x14\x14\x14\x15\x15\x15\x15\x15\x15\x15\x16\x16\x16\x16\x16\x16\x16\x17\x17\x17\x17\x17\x17\x17\x18\x18\x18\x18\x18\x18\x18\x19\x19\x19\x19\x19\x19\x19\x20\x20\x20\x20\x20\x20\x20\x21\x21\x21\x21\x21\x21\x21\x22\x22\x22\x22\x22\x22\x22\x23\x23\x23\x23\x23\x23\x23\x24\x24\x24\x24\x24\x24\x24\x25\x25\x25\x25\x25\x25\x25\x26\x26\x26\x26\x26\x26\x26\x27\x27\x27\x27\x27\x27\x27\x28\x28\x28\x28\x28\x28\x28\x29\x29\x29\x29\x29\x29\x29\x30\x30\x30\x30\x30\x30\x30\x31\x31\x31\x31\x31\x31\x31\x32\x32\x32\x32\x32\x32\x32\x33\x33\x33\x33\x33\x33\x33\x34\x34\x34\x34\x34\x34\x34\x35\x35\x35\x35\x35\x35\x35\x36\x36\x36\x36\x36\x36\x36\x37\x37\x37"

func uvarintLen(u uint32) int {
	return int(uvarintLens[byte(bits.Len32(u))])
}

func appendVarint(dst []byte, i int32) []byte {
	return appendUvarint(dst, uint32(i)<<1^uint32(i>>31))
}

func appendUvarint(dst []byte, u uint32) []byte {
	switch uvarintLen(u) {
	case 5:
		return append(dst,
			byte(u&0x7f|0x80),
			byte((u>>7)&0x7f|0x80),
			byte((u>>14)&0x7f|0x80),
			byte((u>>21)&0x7f|0x80),
			byte(u>>28))
	case 4:
		return append(dst,
			byte(u&0x7f|0x80),
			byte((u>>7)&0x7f|0x80),
			byte((u>>14)&0x7f|0x80),
			byte(u>>21))
	case 3:
		return append(dst,
			byte(u&0x7f|0x80),
			byte((u>>7)&0x7f|0x80),
			byte(u>>14))
	case 2:
		return append(dst,
			byte(u&0x7f|0x80),
			byte(u>>7))
	default:
		return append(dst, byte(u))
	}
}

// appendVarlong writes i as a zigzag-encoded varlong (1-10 bytes).
// Mirrors appendUvarint's length-keyed switch shape: bench shows a
// 30-50% improvement over the generic loop across all lengths
// (BenchmarkVarlongPerLen_*; switch wins at every byte length).
func appendVarlong(dst []byte, i int64) []byte {
	u := uint64(i)<<1 ^ uint64(i>>63)
	switch {
	case u < 1<<7:
		return append(dst, byte(u))
	case u < 1<<14:
		return append(dst, byte(u)|0x80, byte(u>>7))
	case u < 1<<21:
		return append(dst, byte(u)|0x80, byte(u>>7)|0x80, byte(u>>14))
	case u < 1<<28:
		return append(dst, byte(u)|0x80, byte(u>>7)|0x80, byte(u>>14)|0x80, byte(u>>21))
	case u < 1<<35:
		return append(dst, byte(u)|0x80, byte(u>>7)|0x80, byte(u>>14)|0x80, byte(u>>21)|0x80, byte(u>>28))
	case u < 1<<42:
		return append(dst, byte(u)|0x80, byte(u>>7)|0x80, byte(u>>14)|0x80, byte(u>>21)|0x80, byte(u>>28)|0x80, byte(u>>35))
	case u < 1<<49:
		return append(dst, byte(u)|0x80, byte(u>>7)|0x80, byte(u>>14)|0x80, byte(u>>21)|0x80, byte(u>>28)|0x80, byte(u>>35)|0x80, byte(u>>42))
	case u < 1<<56:
		return append(dst, byte(u)|0x80, byte(u>>7)|0x80, byte(u>>14)|0x80, byte(u>>21)|0x80, byte(u>>28)|0x80, byte(u>>35)|0x80, byte(u>>42)|0x80, byte(u>>49))
	case u < 1<<63:
		return append(dst, byte(u)|0x80, byte(u>>7)|0x80, byte(u>>14)|0x80, byte(u>>21)|0x80, byte(u>>28)|0x80, byte(u>>35)|0x80, byte(u>>42)|0x80, byte(u>>49)|0x80, byte(u>>56))
	default:
		return append(dst, byte(u)|0x80, byte(u>>7)|0x80, byte(u>>14)|0x80, byte(u>>21)|0x80, byte(u>>28)|0x80, byte(u>>35)|0x80, byte(u>>42)|0x80, byte(u>>49)|0x80, byte(u>>56)|0x80, byte(u>>63))
	}
}

func readUvarint(src []byte) (uint32, []byte, error) {
	var u uint32
	for i := range 4 {
		if i >= len(src) {
			return 0, nil, &ShortBufferError{Type: "uvarint"}
		}
		b := src[i]
		u |= uint32(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return u, src[i+1:], nil
		}
	}
	if len(src) < 5 {
		return 0, nil, &ShortBufferError{Type: "uvarint"}
	}
	b := src[4]
	if b > 0x0f {
		return 0, nil, errors.New("uvarint overflows 32 bits")
	}
	u |= uint32(b) << 28
	return u, src[5:], nil
}

func readVarint(src []byte) (int32, []byte, error) {
	if len(src) > 0 && src[0] < 0x80 {
		u := uint32(src[0])
		return int32(u>>1) ^ -int32(u&1), src[1:], nil
	}
	u, src, err := readUvarint(src)
	if err != nil {
		return 0, nil, err
	}
	return int32(u>>1) ^ -int32(u&1), src, nil
}

func readUvarlong(src []byte) (uint64, []byte, error) {
	var u uint64
	for i := range 9 {
		if i >= len(src) {
			return 0, nil, &ShortBufferError{Type: "uvarlong"}
		}
		b := src[i]
		u |= uint64(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return u, src[i+1:], nil
		}
	}
	if len(src) < 10 {
		return 0, nil, &ShortBufferError{Type: "uvarlong"}
	}
	b := src[9]
	if b > 0x01 {
		return 0, nil, errors.New("uvarlong overflows 64 bits")
	}
	u |= uint64(b) << 63
	return u, src[10:], nil
}

func readVarlong(src []byte) (int64, []byte, error) {
	if len(src) > 0 && src[0] < 0x80 {
		u := uint64(src[0])
		return int64(u>>1) ^ -int64(u&1), src[1:], nil
	}
	u, src, err := readUvarlong(src)
	if err != nil {
		return 0, nil, err
	}
	return int64(u>>1) ^ -int64(u&1), src, nil
}

func readUint32(src []byte) (uint32, []byte, error) {
	if len(src) < 4 {
		return 0, nil, &ShortBufferError{Type: "uint32", Need: 4, Have: len(src)}
	}
	u := uint32(src[0]) | uint32(src[1])<<8 | uint32(src[2])<<16 | uint32(src[3])<<24
	return u, src[4:], nil
}

func readUint64(src []byte) (uint64, []byte, error) {
	if len(src) < 8 {
		return 0, nil, &ShortBufferError{Type: "uint64", Need: 8, Have: len(src)}
	}
	u := uint64(src[0]) | uint64(src[1])<<8 | uint64(src[2])<<16 | uint64(src[3])<<24 |
		uint64(src[4])<<32 | uint64(src[5])<<40 | uint64(src[6])<<48 | uint64(src[7])<<56
	return u, src[8:], nil
}
