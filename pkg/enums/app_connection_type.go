//go:generate go run github.com/dmarkham/enumer -trimprefix=AppConnectionType -type=AppConnectionType -json -gqlgen -sql -text -transform=snake

package enums

type AppConnectionType int

const (
	AppConnectionTypeServerless AppConnectionType = iota
	AppConnectionTypeConnect
)
