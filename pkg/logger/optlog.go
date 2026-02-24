package logger

import (
	"github.com/google/uuid"
)

// LogEnabler is a function type that will only allow logs for the given account/log name
// on a truthy return value.
type LogEnabler func(acctID uuid.UUID, logname string) bool

// DefaultLogEnabler always disables all optional logging.  Update this package variable
// with your own [LogEnabler] to allow optional logging.
var DefaultLogEnabler = func(acctID uuid.UUID, logname string) bool {
	return false
}
