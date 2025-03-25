//go:generate go run github.com/dmarkham/enumer -trimprefix=TraceRunOrder -type=TraceRunOrder -json -text
package enums

type TraceRunOrder int

const (
	TraceRunOrderDesc TraceRunOrder = iota
	TraceRunOrderAsc
)
