//go:generate go run github.com/dmarkham/enumer -trimprefix=SingletonMode -type=SingletonMode -transform=snake -json -text -gqlgen

package enums

type SingletonMode int

const (
	// SingletonModeSkip skips the new run if another singleton instance is already in progress.
	SingletonModeSkip SingletonMode = iota

	// SingletonModeCancel cancels the currently running singleton instance and starts the new one.
	SingletonModeCancel

	// SingletonModeJoin skips the new run like SingletonModeSkip, but when the
	// new run was triggered by step.invoke, the invoking run is resolved with
	// the output (or error) of the active singleton run instead of waiting for
	// its invoke to time out.
	SingletonModeJoin
)
