package inngestgo

import "github.com/inngest/inngestgo/errors"

type StepError = errors.StepError

// Re-export internal errors for users
var NoRetryError = errors.NoRetryError
var RetryAtError = errors.RetryAtError
