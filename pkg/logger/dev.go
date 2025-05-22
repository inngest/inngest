package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"strings"

	"github.com/fatih/color"
)

// newDevHandler constructs a dev handler to be used
func newDevHandler(w io.Writer, opts *slog.HandlerOptions) *devHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{
			Level: LevelInfo,
		}
	}

	return &devHandler{
		Handler: slog.NewJSONHandler(w, opts),
		writer:  w,
		opts:    opts,
	}
}

// devHandler is used for development purposes and also provide nicer log output for the dev server
type devHandler struct {
	slog.Handler

	writer io.Writer
	opts   *slog.HandlerOptions
	l      *log.Logger
}

func (d *devHandler) Handle(ctx context.Context, rec slog.Record) error {
	var lvl string

	switch rec.Level {
	case LevelTrace:
		lvl = color.CyanString("TRACE")
	case LevelDebug:
		lvl = color.GreenString("DEBUG")
	case LevelInfo:
		lvl = color.BlueString("INFO")
	case LevelWarning:
		lvl = color.YellowString("WARN")
	case LevelError:
		lvl = color.RedString("ERROR")
	case LevelEmergency:
		lvl = color.HiRedString("URGENT")
	}

	d.l.Println(
		rec.Time.Format("[15:05:05.000]"), // timestamp
		lvl,
		rec.Message,
		d.attrToStr(rec),
	)

	return nil
}

func (d *devHandler) attrToStr(rec slog.Record) string {
	var builder strings.Builder
	_, _ = builder.WriteString(" ")

	rec.Attrs(func(a slog.Attr) bool {
		_, _ = builder.WriteString(fmt.Sprintf("%s=%+v", a.Key, a.Value.Any()))
		return true
	})

	return builder.String()
}
