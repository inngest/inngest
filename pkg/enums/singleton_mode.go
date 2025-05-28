//go:generate go run github.com/dmarkham/enumer -trimprefix=SingletonMode -type=SingletonMode -transform=snake -json -text -gqlgen

package enums

type SingletonMode int

const (
	// SingletonModeSkip skips the new run if another singleton instance is already in progress.
	SingletonModeSkip SingletonMode = iota

	// SingletonModeCancel cancels the currently running singleton instance and starts the new one.
	SingletonModeCancel
)
