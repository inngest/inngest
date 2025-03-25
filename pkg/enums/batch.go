//go:generate go run github.com/dmarkham/enumer -trimprefix=Batch -type=Batch -json -text

package enums

type Batch int

const (
	// BatchAppend represents an item being appended to an existing batch
	BatchAppend Batch = iota
	// BatchNew represents a newly created batch
	BatchNew
	// BatchFull represents a full batch
	BatchFull
)
