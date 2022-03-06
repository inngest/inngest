package tests

import (
	defs "inngest.com/defs/v1"
)

minimal: defs.#Function & {
	name: "test"
	triggers: [{
		event: "test.event"
	}]
}

expression: defs.#Function & {
	name: "test"
	triggers: [{
		event:      "test.event"
		expression: "data.run == true"
	}]
}
