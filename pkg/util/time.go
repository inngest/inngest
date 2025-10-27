package util

import (
	"context"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/logger"
)

func AnyToTime(v any) time.Time {
	l := logger.StdlibLogger(context.Background()).With(
		"function", "anyToTime",
		"value", v,
	)
	if v == nil {
		l.Error("value is nil")
		return time.Time{}
	}

	switch v := v.(type) {
	case string:
		st := strings.Split(v, " m=")[0]
		parsedStartTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", st)
		if err != nil {
			l.Error("error parsing time", "error", err)
			return time.Time{}
		}
		return parsedStartTime
	case time.Time:
		return v
	default:
		l.Error("unsupported type")
		return time.Time{}
	}
}
