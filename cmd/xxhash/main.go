package main

import (
	"fmt"

	"github.com/inngest/inngest/pkg/util"
)

func main() {
	input := "01KH1J3YBCYZ3RYA9RA9FAR4AB"
	hash := util.XXHash(input)
	fmt.Printf("Input:  %s\n", input)
	fmt.Printf("XXHash: %s\n", hash)
}
