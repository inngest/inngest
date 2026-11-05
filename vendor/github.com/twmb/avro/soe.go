package avro

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// soeHeader returns the schema's Single Object Encoding header, hashing the
// canonical form on the first call and reusing the result forever after.
// Every read of s.soe goes through here: the SOE entry points are the only
// consumers, so a schema that never writes or reads single-object bytes never
// pays for the hash, and a bare field read would race the first caller that
// does.
//
// The soeHashed guard rather than a bare soeOnce.Do: every single-object
// message pays this check, and inlining sync.Once.Do puts the function 1 unit
// over the inline budget, which cost a measurable 4ns per AppendSingleObject
// on a small payload. The flag is written INSIDE the Once, after the header
// bytes, so a reader that observes it true is ordered after those writes.
func (s *Schema) soeHeader() *[10]byte {
	if !s.soeHashed.Load() {
		s.hashSOEHeader()
	}
	return &s.soe
}

func (s *Schema) hashSOEHeader() {
	s.soeOnce.Do(func() {
		s.soe[0] = 0xC3
		s.soe[1] = 0x01
		h := NewRabin()
		h.Write(s.Canonical())
		binary.LittleEndian.PutUint64(s.soe[2:], h.Sum64())
		s.soeHashed.Store(true)
	})
}

// AppendSingleObject appends a Single Object Encoding of v to dst: 2-byte
// magic, 8-byte CRC-64-AVRO fingerprint, then the Avro binary payload.
func (s *Schema) AppendSingleObject(dst []byte, v any, opts ...Opt) ([]byte, error) {
	dst = append(dst, s.soeHeader()[:]...)
	return s.AppendEncode(dst, v, opts...)
}

// validateSOEHeader checks that data has the 10-byte Single Object Encoding
// header (length + magic bytes). Shared by DecodeSingleObject and
// SingleObjectFingerprint so the two paths agree on the error shapes.
func validateSOEHeader(data []byte) error {
	if len(data) < 10 {
		return fmt.Errorf("avro: single-object encoding too short: need at least 10 bytes, have %d", len(data))
	}
	if data[0] != 0xC3 || data[1] != 0x01 {
		return fmt.Errorf("avro: invalid single-object encoding magic: got [%#x, %#x], want [0xc3, 0x01]", data[0], data[1])
	}
	return nil
}

// DecodeSingleObject decodes a Single Object Encoding message into v after
// verifying the magic and fingerprint match this schema.
//
// For a schema returned by [Resolve], the writer's fingerprint is also
// accepted — wire bytes carry the writer's fingerprint per the SOE spec,
// and a resolved schema is the right place to decode them.
func (s *Schema) DecodeSingleObject(data []byte, v any, opts ...Opt) ([]byte, error) {
	if err := validateSOEHeader(data); err != nil {
		return nil, err
	}
	header := [10]byte(data[:10])
	if header != *s.soeHeader() && !s.acceptsWriterSOE(header) {
		return nil, errors.New("avro: single-object encoding fingerprint mismatch")
	}
	return s.Decode(data[10:], v, opts...)
}

// acceptsWriterSOE reports whether header is the fingerprint of the writer
// this schema resolves from.
//
// A single-object message carries the fingerprint of the schema that PRODUCED
// the bytes, which is the writer whenever a resolution is involved, so a
// resolved schema is the one place those bytes can be decoded. Java reaches the
// same outcome through a fingerprint registry (BinaryMessageDecoder); the
// single-schema model bakes it into the resolved schema's own check.
//
// The writer is reached through the writer SCHEMA rather than a copied header,
// which keeps its fingerprint lazy: a resolved schema that never decodes
// single-object bytes hashes neither canonical form. It asks resolveWriter —
// the one field holding that schema — so writer acceptance cannot drift from
// the resolution it belongs to. Always false for a schema that is not a
// resolution, which has no writer to accept; resolveWriter is nil for exactly
// that set, since an identity resolution returns the reader itself.
func (s *Schema) acceptsWriterSOE(header [10]byte) bool {
	return s.resolveWriter != nil && header == *s.resolveWriter.soeHeader()
}

// SingleObjectFingerprint extracts the 8-byte CRC-64-AVRO fingerprint and
// returns the remaining payload.
func SingleObjectFingerprint(data []byte) (fp [8]byte, rest []byte, err error) {
	if err := validateSOEHeader(data); err != nil {
		return fp, nil, err
	}
	copy(fp[:], data[2:10])
	return fp, data[10:], nil
}
