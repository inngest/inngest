package httpv2

import (
	"time"

	"github.com/inngest/inngest/pkg/util/errs"
)

func NewNonGeneratorError(out []byte, status int) errs.UserError {
	return NonGeneratorError{Response: out, Status: status}
}

// NonGeneratorError is a UserError which indicates that the SDK didn't return
// opcodes as a response.  In new modes of the SDK, everything should be a
// generator.
type NonGeneratorError struct {
	Response []byte
	Status   int
}

func (n NonGeneratorError) UserError() {}

func (n NonGeneratorError) Retryable() bool { return n.Status > 299 }

func (n NonGeneratorError) RetryAfter() time.Duration { return 0 }

func (n NonGeneratorError) ErrorCode() int {
	// todo
	return 0
}

func (n NonGeneratorError) Raw() []byte {
	return n.Response
}

func (n NonGeneratorError) Error() string {
	return "SDK returned an unexpected response"
}
