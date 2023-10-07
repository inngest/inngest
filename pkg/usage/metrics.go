package usage

import (
	"fmt"
	"time"
)

var (
	NoMetricsNameErr           = fmt.Errorf("metrics' name must be specified")
	NoMetricsTimeRangeErr      = fmt.Errorf("metrics' time range (from/to - ISO8601 format) must be specified")
	InvalidMetricsTimeRangeErr = fmt.Errorf("invalid time range for metrics")
)

// MetricsRequest represents a client request for metrics based on time range
type MetricsRequest struct {
	Name string    `json:"name"`
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func (mr MetricsRequest) Valid() error {
	if mr.Name == "" {
		return NoMetricsNameErr
	}

	if mr.From.IsZero() || mr.To.IsZero() {
		return NoMetricsTimeRangeErr
	}

	if mr.To.Sub(mr.From) < 0 {
		return InvalidMetricsTimeRangeErr
	}

	return nil
}

// Granularity returns the predefined aggregation period for
// the query
func (mr MetricsRequest) Granularity() time.Duration {
	dur := mr.To.Sub(mr.From)
	day := 24 * time.Hour

	switch {
	case dur >= 30*day:
		return 12 * time.Hour
	case dur >= 14*day:
		return 6 * time.Hour
	case dur >= 7*day:
		return 3 * time.Hour
	case dur >= 3*day:
		return time.Hour
	case dur >= 1*day:
		return 30 * time.Minute
	case dur >= 12*time.Hour:
		return 15 * time.Minute
	case dur >= 6*time.Hour:
		return 10 * time.Minute
	case dur >= 3*time.Hour:
		return 5 * time.Minute
	default:
		return time.Minute
	}
}

// MetricsResponse represents an API response to a MetricsRequest
type MetricsResponse struct {
	Name        string        `json:"name"`
	From        time.Time     `json:"from"`
	To          time.Time     `json:"to"`
	Granularity string        `json:"granularity"`
	Data        []MetricsData `json:"data"`
}

// MetricsData represents a single slot of timeseries data.
type MetricsData struct {
	Bucket time.Time `json:"bucket"`
	Value  float64   `json:"value"`
}
