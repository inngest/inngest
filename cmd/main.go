package main

import (
	_ "time/tzdata" // bundles timezone data, required for Windows without Go

	"github.com/inngest/inngest/cmd/commands"
)

func main() {
	commands.Execute()
}
