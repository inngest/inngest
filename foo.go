package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func main() {
	original := `"hi"`
	reader := strings.NewReader(original)
	decoder := json.NewDecoder(reader)
	_, _ = decoder.Token()

	rest, _ := io.ReadAll(decoder.Buffered())
	fmt.Println(string(rest) == original)
}
