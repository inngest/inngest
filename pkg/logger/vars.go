package logger

import (
	"fmt"
	"os"
)

var (
	host = ""
)

// initialize global variables that will be referenced
func init() {
	h, err := os.Hostname()
	if err != nil {
		panic(fmt.Errorf("error retriving hostname: %w", err))
	}
	host = h
}
