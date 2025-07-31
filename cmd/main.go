package main

import (
	"os"
	"time"
	_ "time/tzdata" // bundles timezone data, required for Windows without Go
)

func main() {
	// Ensure that we use UTC everywhere. This is a fix for users getting
	// invalid timestamps due to their specific `/etc/localtime` files.
	if err := os.Setenv("TZ", "UTC"); err != nil {
		panic(err)
	}
	time.Local = time.UTC

	execute()
}
