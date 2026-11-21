// Package signingkey derives the bearer token that authenticates calls to the
// Inngest API from a signing key.
package signingkey

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
)

var prefixRegexp = regexp.MustCompile(`^signkey-\w+-`)

// Hash returns the bearer token for a signing key.  the token keeps the
// "signkey-<env>-" prefix and replaces the hex secret with its SHA-256 hash.
// it fails when the secret is not hex.
func Hash(key []byte) ([]byte, error) {
	prefix := prefixRegexp.Find(key)
	key = prefixRegexp.ReplaceAll(key, []byte{})

	dst := make([]byte, hex.DecodedLen(len(key)))
	if _, err := hex.Decode(dst, key); err != nil {
		return nil, err
	}

	sum := sha256.Sum256(dst)
	enc := hex.EncodeToString(sum[:])
	return append(prefix, []byte(enc)...), nil
}
