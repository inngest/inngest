package main

import (
	"fmt"
	"os"

	"github.com/inngest/inngestgo/actionsdk"
)

func main() {
	actionsdk.WriteError(fmt.Errorf("fake error to retry"), true)
	os.Exit(1)
}
