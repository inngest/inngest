package main

import (
	"fmt"
	"os"

	"github.com/inngest/inngest/pkg/util"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: xxhash <input>\n")
		os.Exit(1)
	}
	input := os.Args[1]
	hash := util.XXHash(input)
	fmt.Printf("Input:  %s\n", input)
	fmt.Printf("XXHash: %s\n", hash)
}
