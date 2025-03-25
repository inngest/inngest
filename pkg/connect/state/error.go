package state

import (
	"fmt"
)

var (
	ErrIdempotencyKeyExists = fmt.Errorf("idempotency key exists")
)
