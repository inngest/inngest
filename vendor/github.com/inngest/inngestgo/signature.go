package inngestgo

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

var (
	ErrExpiredSignature = fmt.Errorf("expired signature")
	ErrInvalidSignature = fmt.Errorf("invalid signature")
	ErrInvalidTimestamp = fmt.Errorf("invalid timestamp")

	keyRegexp             = regexp.MustCompile(`^signkey-\w+-`)
	signatureTimeDeltaMax = 5 * time.Minute
)

// Sign signs a request body with the given key at the given timestamp.
func Sign(ctx context.Context, at time.Time, key, body []byte) string {
	key = normalizeKey(key)

	ts := at.Unix()
	if at.IsZero() {
		ts = time.Now().Unix()
	}

	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(body)
	// Write the timestamp as a unix timestamp to the hmac to prevent
	// timing attacks.
	_, _ = mac.Write([]byte(fmt.Sprintf("%d", ts)))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d&s=%s", ts, sig)
}

// ValidateSignature ensures that the signature for the given body is signed with
// the given key recently.
func ValidateSignature(ctx context.Context, sig string, key, body []byte) (bool, error) {
	key = normalizeKey(key)

	val, err := url.ParseQuery(sig)
	if err != nil || (val.Get("t") == "" || val.Get("s") == "") {
		return false, ErrInvalidSignature
	}
	str, err := strconv.Atoi(val.Get("t"))
	if err != nil {
		return false, ErrInvalidTimestamp
	}
	ts := time.Unix(int64(str), 0)
	if time.Since(ts) > signatureTimeDeltaMax {
		return false, ErrExpiredSignature
	}

	actual := Sign(ctx, ts, key, body)
	if actual != sig {
		return false, ErrInvalidSignature
	}

	return true, nil
}

func normalizeKey(key []byte) []byte {
	return keyRegexp.ReplaceAll(key, []byte{})
}

func hashedSigningKey(key []byte) ([]byte, error) {
	prefix := keyRegexp.Find(key)
	key = normalizeKey(key)

	dst := make([]byte, hex.DecodedLen(len(key)))
	if _, err := hex.Decode(dst, key); err != nil {
		return nil, err
	}

	sum := sha256.Sum256(dst)
	enc := hex.EncodeToString(sum[:])
	return append(prefix, []byte(enc)...), nil
}
