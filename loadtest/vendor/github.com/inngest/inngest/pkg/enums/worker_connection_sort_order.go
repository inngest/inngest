//go:generate go run github.com/dmarkham/enumer -trimprefix=WorkerConnectionSortOrder -type=WorkerConnectionSortOrder -json -text
package enums

type WorkerConnectionSortOrder int

const (
	WorkerConnectionSortOrderDesc WorkerConnectionSortOrder = iota
	WorkerConnectionSortOrderAsc
)
