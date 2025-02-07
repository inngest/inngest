//go:generate go run github.com/dmarkham/enumer -trimprefix=AppMethod -type=AppMethod -json -gqlgen -sql -text -transform=snake

package enums

type AppMethod int

const (
	AppMethodServe AppMethod = iota
	AppMethodConnect
)
