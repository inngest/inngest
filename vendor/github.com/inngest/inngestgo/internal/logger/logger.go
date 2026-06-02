package logger

import (
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/lmittmann/tint"
)

const (
	// LevelTrace matches the common slog convention of placing trace below debug.
	LevelTrace slog.Level = -8
)

// Default returns the SDK's default slog logger, configured from environment
// variables.
func Default() *slog.Logger {
	return New(os.Stderr)
}

// New returns a slog logger configured from environment variables and writing
// to w. It is exported for tests and internal packages that need custom writers.
func New(w io.Writer) *slog.Logger {
	handlerOpts := &slog.HandlerOptions{
		Level: parseLevel(os.Getenv("LOG_LEVEL")),
	}

	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_HANDLER"))) {
	case "json":
		return slog.New(slog.NewJSONHandler(w, handlerOpts))
	case "txt", "text":
		return slog.New(slog.NewTextHandler(w, handlerOpts))
	case "dev", "":
		return slog.New(devHandler(w, handlerOpts.Level))
	default:
		return slog.New(devHandler(w, handlerOpts.Level))
	}
}

func devHandler(w io.Writer, level slog.Leveler) slog.Handler {
	return tint.NewHandler(w, &tint.Options{
		Level:      level,
		TimeFormat: "[15:04:05.000]",
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.LevelKey && len(groups) == 0 {
				lvl, ok := attr.Value.Any().(slog.Level)
				if ok {
					switch lvl {
					case LevelTrace:
						return tint.Attr(13, slog.String(attr.Key, "TRC"))
					case slog.LevelDebug:
						return tint.Attr(3, slog.String(attr.Key, "DBG"))
					case slog.LevelInfo:
						return tint.Attr(14, slog.String(attr.Key, "INF"))
					}
				}
			}
			return attr
		},
	})
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "trace":
		return LevelTrace
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		if n, err := strconv.Atoi(value); err == nil {
			return slog.Level(n)
		}
		return slog.LevelInfo
	}
}
