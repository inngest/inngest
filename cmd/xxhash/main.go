package main

import (
	"fmt"

	"github.com/inngest/inngest/pkg/util"
)

func main() {
	input := "72a2aaf4-808b-4b24-b1b6-bf3883c9d559"
	fmt.Printf("Input:  %s\n", input)
	fmt.Printf("XXHash: %s\n", util.XXHash(input))
}
