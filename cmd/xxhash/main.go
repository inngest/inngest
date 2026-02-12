package main

import (
	"fmt"

	"github.com/inngest/inngest/pkg/util"
)

func main() {
	input := "1edd0832-b21a-4e76-a604-41a2041832b3"
	fmt.Printf("Input:  %s\n", input)
	fmt.Printf("XXHash: %s\n", util.XXHash(input))
}
