package inngestgo

import "github.com/inngest/inngestgo/errors"

// Re-export internal errors for users
var NoRetryError = errors.NoRetryError
var RetryAtError = errors.RetryAtError
