//go:generate go run github.com/dmarkham/enumer -trimprefix=ParallelMode -type=ParallelMode -json -text

package enums

type ParallelMode int

const (
	// ParallelModeNone defaults to ParallelModeWait
	ParallelModeNone ParallelMode = iota

	// ParallelModeWait will wait for all parallel steps to end before
	// scheduling a "discovery request". This can significantly reduce the
	// number of requests sent to your SDK. However, it doesn't allow
	// "sequential steps" (i.e. more than 1 step) in parallel groups to run
	// independently.
	ParallelModeWait

	// ParallelModeRace will schedule a "discovery request" immediately after
	// each parallel step ends. This allows "sequential steps" (i.e. more than 1
	// step) in parallel groups to run independently. However, it can
	// significantly increase the number of requests sent to your SDK. Only use
	// this if you have more than 1 step in a parallel group and you want it to
	// run independently of the other parallel groups.
	ParallelModeRace
)
