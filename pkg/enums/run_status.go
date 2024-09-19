//go:generate go run github.com/dmarkham/enumer -trimprefix=RunStatus -type=RunStatus -json -text -gqlgen

package enums

import (
	"fmt"
	"math"
	"strconv"
)

type RunStatus int

// NOTE:
// DO NOT EVER DELETE OR REUSE.
// There are Lua scripts that rely on the integer values in the state metadata.
// Deleting/reusing enum value will break things.
//
//goland:noinspection GoDeprecation
const (
	// RunStatusRunning indicates that the function is running.  This is the
	// default state, even if steps are scheduled in the future.
	RunStatusRunning RunStatus = 0
	// RunStatusCompleted indicates that the function has completed running.
	RunStatusCompleted RunStatus = 1
	// RunStatusFailed indicates that the function failed in one or more steps.
	RunStatusFailed RunStatus = 2
	// RunStatusCancelled indicates that the function has been cancelled prior
	// to any errors
	RunStatusCancelled RunStatus = 3
	// RunStatusOverflowed indicates that the function had too many steps ran.
	// Deprecated.  This must be RunStatusFailed with an appropriate error code.
	RunStatusOverflowed RunStatus = 4
	// RunStatusScheduled indicates that the function is scheduled but have not started
	// processing
	RunStatusScheduled RunStatus = 5
	// RunStatusUnknown indicates that the function is in an unknown status.
	// This is unlikely to happen during normal execution, and more likely when converting between
	// the status code
	RunStatusUnknown RunStatus = 6
	// RunStatusSkipped indicates that the function was skipped and not ran
	RunStatusSkipped RunStatus = 7
)

var (
	// NOTE: assign larger number status codes RunStatus
	// This can be used for cases where you need the numbers to be in ascending order based on status
	runStatusCode = map[RunStatus]int64{
		RunStatusOverflowed: 50,
		RunStatusScheduled:  100,
		RunStatusRunning:    200,
		RunStatusCompleted:  300,
		RunStatusFailed:     400,
		RunStatusCancelled:  500,
		RunStatusSkipped:    600,
	}

	codeStatusMap = map[int64]RunStatus{}
)

func init() {
	// reverse the map for look up
	for k, v := range runStatusCode {
		codeStatusMap[v] = k
	}
}

// RunStatusEnded returns whether the function has ended based off of the
// run status.
func RunStatusEnded(s RunStatus) bool {
	if s == RunStatusCancelled || s == RunStatusCompleted || s == RunStatusFailed || s == RunStatusOverflowed || s == RunStatusSkipped {
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
	if i < 0 || i > math.MaxInt {
		return fmt.Errorf("enum value is out of bound of int type: %d", i)
	}

	rs := RunStatus(int(i))
	*r = rs
	return nil
}

func (r RunStatus) ToCode() int64 {
	if code, ok := runStatusCode[r]; ok {
		return code
	}

	return 0 // unknown
}

func RunCodeToStatus(val int64) RunStatus {
	if status, ok := codeStatusMap[val]; ok {
		return status
	}
	return RunStatusUnknown
}
