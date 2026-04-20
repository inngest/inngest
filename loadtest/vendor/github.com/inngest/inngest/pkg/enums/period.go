//go:generate go run github.com/dmarkham/enumer -trimprefix=Period -type=Period -json -gqlgen -sql -text -transform=snake

package enums

type Period int

const (
	PeriodNone Period = iota
	PeriodMinute
	PeriodHour
	PeriodDay
	PeriodWeek
	PeriodMonth
)

func DefaultRange(p Period) Timerange {
	switch p {
	case PeriodMinute:
		return TimerangeHour
	case PeriodHour:
		return TimerangeDay
	case PeriodDay:
		return TimerangeWeek
	case PeriodWeek:
		return TimerangeMonth
	case PeriodMonth:
		return TimerangeYear
	}
	return TimerangeDay
}

func MaxRange(p Period) Timerange {
	switch p {
	case PeriodMinute:
		return TimerangeHour
	case PeriodHour:
		return TimerangeDay
	case PeriodDay:
		return TimerangeMonth
	case PeriodWeek:
		return TimerangeYear
	case PeriodMonth:
		return TimerangeYear
	}
	return TimerangeDay
}
