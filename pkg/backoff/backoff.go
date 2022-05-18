package backoff

import (
	"time"

	"math/rand"
)

func LinearJitterBackoff(attemptNum int) time.Time {
	backoff := float64(uint(1) << (uint(attemptNum) - 1))
	backoff += backoff * (0.15 * rand.Float64())
	// Increase by a factor of 10 to get 10 second breaks at minimum.
	backoff = backoff * 10
	return time.Now().Add(time.Second * time.Duration(backoff))
}
