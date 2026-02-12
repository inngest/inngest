package main

import (
	"fmt"

	"github.com/inngest/inngest/pkg/util"
)

func main() {
	// Hash the full string
	input1 := "f:505b10bd-548b-4436-baf3-6dd06311cd7d:1edd0832-b21a-4e76-a604-41a2041832b3"
	fmt.Printf("Input:  %s\n", input1)
	fmt.Printf("XXHash: %s\n\n", util.XXHash(input1))

	// Hash with the last segment pre-hashed
	hashedUUID := util.XXHash("1edd0832-b21a-4e76-a604-41a2041832b3")
	input2 := "f:505b10bd-548b-4436-baf3-6dd06311cd7d:" + hashedUUID
	fmt.Printf("Input:  %s\n", input2)
	fmt.Printf("XXHash: %s\n", util.XXHash(input2))
}
