//go:generate go run github.com/dmarkham/enumer -trimprefix=TraceRunTime -type=TraceRunTime -transform=snake -json -text
package enums

type RunMode int

const (
	RunModeAsync RunMode = iota
	RunModeSync
)
