//go:generate go run github.com/dmarkham/enumer -trimprefix=TraceRunOrder -type=TraceRunOrder -json -text
package enums

type WorkerConnectionSortOrder int

const (
	WorkerConnectionSortOrderDesc WorkerConnectionSortOrder = iota
	WorkerConnectionSortOrderAsc
)
