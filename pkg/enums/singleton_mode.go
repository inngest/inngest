//go:generate go run github.com/dmarkham/enumer -trimprefix=SingletonMode -type=SingletonMode -transform=snake -json -text -gqlgen

package enums

type SingletonMode int

const (
	// SingletonModeSkip skips the new run if another singleton instance is already in progress.
	SingletonModeSkip SingletonMode = iota

	// SingletonModeCancel cancels the currently running singleton instance and starts the new one.
	SingletonModeCancel

	// SingletonModeQueue hands a new event to the running instance's "singleton" inbox,
	// or holds it for the next run when that inbox is closed, instead of skipping it.
	SingletonModeQueue
)
