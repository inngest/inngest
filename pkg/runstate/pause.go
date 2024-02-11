package runstate

import (
	"time"

	"github.com/oklog/ulid/v2"
)

type Pause struct {
	ID         ulid.ULID
	Event      string
	Expression *string
	StepName   string
	StepID     StepID
	Opcode     int
	Expires    time.Time
	Timeout    bool
	Cancel     bool
}
