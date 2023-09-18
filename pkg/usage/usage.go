package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/pkg/errors"
)

type UsageInput struct {
	Period *enums.Period    `json:"period"`
	Range  *enums.Timerange `json:"range"`
	From   *time.Time       `json:"from"`
	To     *time.Time       `json:"to"`
}

// Validate validates that the period and range are specified correctly.  A minimum period
// can be supplied to this function;  any periods shorter than the minimum are invalid.
//
// If the min is nil any enums.Period value is valid.
func (u *UsageInput) Validate(ctx context.Context, min *enums.Period) error {
	defaultPeriod := enums.PeriodHour

	if u == nil {
		u = &UsageInput{
			Period: &defaultPeriod,
		}
	}

	if u.Period == nil {
		u.Period = &defaultPeriod
	}
	if u.Range == nil {
		defaultRange := enums.MaxRange(*u.Period)
		u.Range = &defaultRange
	}
	if int(*u.Period) > int(*u.Range) {
		return errors.New("range must be smaller than period")
	}
	if enums.MaxRange(*u.Period) < *u.Range {
		return errors.Errorf("range must be smaller than %s", enums.MaxRange(*u.Period))
	}
	return nil
}

// UsageResponse represents event usage as queried by our event store
type UsageResponse struct {
	// Period is the period that this usage represents
	Period enums.Period    `json:"period"`
	Range  enums.Timerange `json:"range"`

	// Data represents the individual aggregated usage data for a specific point
	// in time.
	Data []UsageSlot `json:"data"`

	// AsOf represents the time that this usage was valid for.  We may store
	// results in a cache, and this allows us to see how stale the data is.
	AsOf time.Time `json:"asOf"`
}

func (u UsageResponse) Total(ctx context.Context) int64 {
	acc := int64(0)
	for _, d := range u.Data {
		acc += d.Count
	}
	return acc
}

func (u *UsageResponse) Add(b UsageResponse) error {
	if len(u.Data) != len(b.Data) {
		return fmt.Errorf("response mismatch")
	}
	for n, slot := range b.Data {
		u.Data[n].Count += slot.Count
	}
	return nil
}

type UsageSlot struct {
	// Slot represents the start time for the slot
	// TODO: use unix timestamp - smaller, faster
	Slot  time.Time `json:"slot"`
	Count int64     `json:"count"`
}
