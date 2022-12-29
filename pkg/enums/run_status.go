//go:generate go run github.com/tonyhb/enumer -trimprefix=RunStatus -type=RunStatus -json -text

package enums

import (
	"strconv"
)

type RunStatus int

const (
	// RunStatusRunning indicates that the function is running.  This is the
	// default state, even if steps are scheduled in the future.
	RunStatusRunning RunStatus = iota
	// RunStatusCompleted indicates that the function has completed running.
	RunStatusCompleted
	// RunStatusFailed indicates that the function failed in one or more steps.
	RunStatusFailed
	// RunStatusCancelled indicates that the function has been cancelled prior
	// to any errors
	RunStatusCancelled
)

// RunStatusEnded returns whether the function has ended based off of the
// run status.
func RunStatusEnded(s RunStatus) bool {
	if s == RunStatusCancelled || s == RunStatusCompleted || s == RunStatusFailed {
		return true
	}
	return false
}

func (r RunStatus) MarshalBinary() ([]byte, error) {
	byt := []byte{}
	return strconv.AppendInt(byt, int64(r), 10), nil
}

func (r *RunStatus) UnmarshalBinary(byt []byte) error {
	i, err := strconv.ParseInt(string(byt), 10, 64)
	if err != nil {
		return err
	}
	rs := RunStatus(i)
	*r = rs
	return nil
}
