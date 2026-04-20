//go:generate go run github.com/dmarkham/enumer -trimprefix=Timerange -type=Timerange -json -gqlgen -sql -text -transform=snake

package enums

type Timerange int

const (
	TimerangeNone Timerange = iota
	TimerangeHour
	TimerangeDay
	TimerangeWeek
	TimerangeMonth
	TimerangeYear
)
