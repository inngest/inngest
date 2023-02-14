package main

import (
	"os"

	"github.com/inngest/inngestgo/actionsdk"
)

func main() {
	// Get the step's input arguments.
	args := actionsdk.MustGetArgs()

	// Write the result of a step function here.  If this errors,
	// you can use actionsdk.WriteError.
	actionsdk.WriteResult(&actionsdk.Result{
		Body: map[string]string{
			"event": args.Event.Name,
			"FOO":   os.Getenv("FOO"),
			"QUOTE": os.Getenv("QUOTE"),
			"CERT":  os.Getenv("CERT"),
		},
		Status: 200,
	})
}
