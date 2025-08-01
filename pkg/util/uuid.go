package util

import "github.com/google/uuid"

// DeterministicUUID returns a deterministic V3 UUID based off of the SHA1
// hash of the input string.
func DeterministicUUID(input []byte) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, input)
}
