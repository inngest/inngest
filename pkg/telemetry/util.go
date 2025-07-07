package telemetry

import (
	"fmt"
	"runtime"
	"strings"
)

func Caller() string {
	skip := 3
	file := ""
	line := 0
	for file == "" || strings.Contains(file, "vendor") || strings.Contains(file, "osql") || strings.Contains(file, "go/src") {
		_, file, line, _ = runtime.Caller(skip)
		skip++
	}

	return fmt.Sprintf("%s:%d", file, line)
}
