package event

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/oklog/ulid/v2"
)

// SeededIDFromString parses an event idempotency key header value and returns a
// new SeededID.
//
// The "value" param must be of the form "millis,entropy", where millis is the
// number of milliseconds since the Unix epoch, and entropy is a base64-encoded
// 10-byte value. For example: "1743130137367,eii2YKXRVTJPuA==".
//
// The "index" param is the index of the event in the request. This is used to
// give each event in a multi-event payload its own unique entropy despite only
// 1 entropy value being in the request header.
func SeededIDFromString(value string, index int) *SeededID {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return nil
	}

	millis, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil
	}
	if millis <= 0 {
		return nil
	}

	entropy, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	if len(entropy) != 10 {
		return nil
	}

	// Add the index to the entropy to allow a single seed string to generate
	// many unique ULIDs.
	binary.BigEndian.PutUint32(
		entropy[6:10],
		binary.BigEndian.Uint32(entropy[6:10])+uint32(index),
	)

	return &SeededID{
		Entropy: entropy,
		Millis:  millis,
	}
}

type SeededID struct {
	// Entropy is the 10-byte entropy value used to generate the ULID.
	Entropy []byte

	// Millis is the number of milliseconds since the Unix epoch.
	Millis int64
}

func (s *SeededID) ToULID() (ulid.ULID, error) {
	if len(s.Entropy) != 10 {
		return ulid.ULID{}, fmt.Errorf("entropy must be 10 bytes")
	}

	if s.Millis <= 0 {
		return ulid.ULID{}, fmt.Errorf("millis must be greater than 0")
	}

	return ulid.New(uint64(s.Millis), bytes.NewReader(s.Entropy))
}
