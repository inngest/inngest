package util

import (
	"flag"
	"os"
)

// InTestMode returns true if the test flag is set, which is injected
// automatically by `go test`.
func InTestMode() bool {
	return flag.Lookup("test.v") != nil || // `go test` targetting a binary
		os.Getenv("TEST_MODE") == "true" // explicit test mode
}
