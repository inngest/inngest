package httpv2

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Sign signs the body with a private key, ensuring that HTTP handlers can verify
// that the request comes from us.
func Sign(ctx context.Context, key, body []byte) string {
	if len(key) == 0 {
		return ""
	}

	now := time.Now().Unix()
	mac := hmac.New(sha256.New, key)

	_, _ = mac.Write(body)
	// Write the timestamp as a unix timestamp to the hmac to prevent
	// timing attacks.
	_, _ = fmt.Fprintf(mac, "%d", now)

	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d&s=%s", now, sig)
}
