package avro

import "hash"

const rabinEmpty = 0xc15d213aa4d7a795

var rabinTable [256]uint64

func init() {
	for i := range 256 {
		fp := uint64(i)
		for range 8 {
			fp = (fp >> 1) ^ (rabinEmpty & -(fp & 1))
		}
		rabinTable[i] = fp
	}
}

type rabin struct{ crc uint64 }

// NewRabin returns a hash.Hash64 computing the CRC-64-AVRO (Rabin)
// fingerprint defined by the Avro specification.
func NewRabin() hash.Hash64 { return &rabin{crc: rabinEmpty} }

func (d *rabin) Write(p []byte) (int, error) {
	for _, b := range p {
		d.crc = (d.crc >> 8) ^ rabinTable[byte(d.crc)^b]
	}
	return len(p), nil
}

func (d *rabin) Sum64() uint64 { return d.crc }

func (d *rabin) Sum(in []byte) []byte {
	s := d.crc
	return append(in, byte(s>>56), byte(s>>48), byte(s>>40), byte(s>>32),
		byte(s>>24), byte(s>>16), byte(s>>8), byte(s))
}

func (d *rabin) Reset()         { d.crc = rabinEmpty }
func (d *rabin) Size() int      { return 8 }
func (d *rabin) BlockSize() int { return 1 }
