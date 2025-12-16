package state

import (
	"errors"

	"github.com/inngest/inngest/pkg/syscode"
)

// IsConnectWorkerAtCapacity checks if the error code indicates that connect workers are at capacity.
// It returns true for both CodeConnectAllWorkersAtCapacity and CodeConnectRequestAssignWorkerReachedCapacity.
func IsConnectWorkerAtCapacityCode(code string) bool {
	return code == syscode.CodeConnectAllWorkersAtCapacity || code == syscode.CodeConnectRequestAssignWorkerReachedCapacity
}

func IsConnectWorkerAtCapacityError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, ErrConnectWorkerCapacity) || IsConnectWorkerAtCapacityCode(err.Error())
}
