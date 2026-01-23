//go:generate go run github.com/dmarkham/enumer -trimprefix=SingletonMode -type=SingletonMode -transform=snake -json -text -gqlgen

package enums

type SingletonMode int

const (
	// SingletonModeSkip skips the new run if another singleton instance is already in progress.
	SingletonModeSkip SingletonMode = iota

	// SingletonModeCancel cancels the currently running singleton instance and starts the new one.
	SingletonModeCancel

	// SingletonModeQueue skips new runs if another singleton is in the queue.  When a run starts,
	// we can enqueue another single run which will take place after the current run.  This essentially
	// is a semaphore to ensure only one instance exists in the queue, but we let another item queue
	// while any function is running.
	SingletonModeQueue
)
