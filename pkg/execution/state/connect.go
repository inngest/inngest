package state

import (
	"errors"
	"math/rand/v2"
	"time"

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

// IsConnectNoHealthyConnectionCode checks if the error code indicates that
// routing found no healthy Connect worker connection (none connected, or all
// draining), so nothing was forwarded to a worker.
func IsConnectNoHealthyConnectionCode(code string) bool {
	return code == syscode.CodeConnectNoHealthyConnection
}

func IsConnectNoHealthyConnectionError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, ErrConnectNoHealthyConnection) || IsConnectNoHealthyConnectionCode(err.Error())
}

// ConnectNoHealthyConnectionRetryAt returns when a run halted for lack of a
// healthy Connect worker connection re-checks routing: the fixed delay plus
// random jitter.
func ConnectNoHealthyConnectionRetryAt(now time.Time) time.Time {
	jitter := time.Duration(rand.Int64N(int64(ConnectNoHealthyConnectionRetryJitter)))
	return now.Add(ConnectNoHealthyConnectionRetryDelay + jitter)
}
