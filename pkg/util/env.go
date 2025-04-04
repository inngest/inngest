package util

import "flag"

// InTestMode returns true if the test flag is set, which is injected
// automatically by `go test`.
func InTestMode() bool {
	return flag.Lookup("test.v") != nil
}
