package logger

import (
	"github.com/google/uuid"
)

type LogEnabler func(acctID uuid.UUID, logname string) bool

var DefaultLogEnabler = func(acctID uuid.UUID, logname string) bool {
	return false
}

func ShouldLog(acctID uuid.UUID, logname string)
