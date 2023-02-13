package main

import (
	"github.com/inngest/inngestgo/actionsdk"
)

func main() {
	// Get the step's input arguments.
	args := actionsdk.MustGetArgs()

	// Write the result of a step function here.  If this errors,
	// you can use actionsdk.WriteError.
	actionsdk.WriteResult(&actionsdk.Result{
		Body:   args.Event.Name,
		Status: 200,
	})
}
