//go:generate go run github.com/dmarkham/enumer -trimprefix=SyncKind -type=SyncKind -json -gqlgen -sql -text -transform=snake

package enums

type SyncKind int

const (
	SyncKindNone SyncKind = iota
	SyncKindInBand
	SyncKindOutOfBand
)
