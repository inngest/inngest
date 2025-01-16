//go:generate go run github.com/dmarkham/enumer -type=AppConnectionType -json -sql -text -gqlgen -transform=snake -trimprefix=AppConnectionType

package enums

type AppConnectionType int

const (
	AppConnectionTypeServerless AppConnectionType = iota
	AppConnectionTypeWorker
)
