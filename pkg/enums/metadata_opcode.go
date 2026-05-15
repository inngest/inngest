//go:generate go run github.com/dmarkham/enumer -trimprefix=MetadataOpcode -type=MetadataOpcode -json -text -transform=snake
package enums

type MetadataOpcode int

const (
	// OpcodeNone represents the default opcode 0, which does nothing
	MetadataOpcodeMerge  MetadataOpcode = iota // Shallowly replace old metadata with new metadata by key
	MetadataOpcodeSet                          // Replace old metadata
	MetadataOpcodeDelete                       // Shallowly delete metadata by key
	MetadataOpcodeAdd                          // Shallowly add numeric metadata keys
)
