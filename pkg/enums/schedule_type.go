//go:generate go run github.com/dmarkham/enumer -trimprefix=ScheduleType -type=ScheduleType -json -text -gqlgen

package enums

type ScheduleType int

const (
	ScheduleTypeUnknown  ScheduleType = iota
	ScheduleTypeEvent
	ScheduleTypeCron
	ScheduleTypeDebounce
	ScheduleTypeRerun
)
