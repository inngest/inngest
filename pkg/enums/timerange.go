//go:generate go run github.com/tonyhb/enumer -trimprefix=Timerange -type=Timerange -json -gql -sql -text -transform=snake

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
