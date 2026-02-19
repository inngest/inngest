package main

import (
	"fmt"

	"github.com/cespare/xxhash/v2"
)

func main() {
	input := "01KH1J3YBCYZ3RYA9RA9FAR4AB"
	hash := xxhash.Sum64String(input)
	fmt.Printf("Input:  %s\n", input)
	fmt.Printf("XXHash: %d\n", hash)
	fmt.Printf("Hex:    %x\n", hash)
}
