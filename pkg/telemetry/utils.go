package telemetry

import (
	"os"
)

func env() string {
	val := os.Getenv("ENV")
	if val == "" {
		val = "development"
	}
	return val
}
