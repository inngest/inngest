//go:generate go run github.com/dmarkham/enumer -trimprefix=DeferStatus -type=DeferStatus -json -text -transform=snake

package enums

type DeferStatus int

const (
	// Unused
	DeferStatusUnknown DeferStatus = iota

	// Already scheduled (when defer is configured to run immediately)
	DeferStatusScheduled

	// Schedule after parent run ends
	DeferStatusAfterRun

	// Will not schedule. Terminal: no transition out.
	DeferStatusAborted

	// Defer was rejected, either because of validation or exceeding a limit.
	// Terminal: no transition out.
	DeferStatusRejected
)
