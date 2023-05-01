package tests

import (
	defs "inngest.com/defs/v1"
)

minimal: defs.#Function & {
	id:   "some-id"
	name: "test"
	triggers: [{
		event: "test.event"
	}]
}

complex: defs.#Function & {
	id:   "some-id"
	name: "test"
	triggers: [{
		event:      "test.event"
		expression: "data.run == true"
	}]
	idempotency: "{{ event.data.foo }}"
	steps: {
		first: {
			runtime: defs.#RuntimeHTTP & {url: "http://www.example.com"}
			after: [{
				step: "$trigger"
				wait: "5m"
			}]
		}
		second: {
			runtime: defs.#RuntimeHTTP & {url: "http://www.example.com"}
			after: [{
				step: "first"
			}]
		}
	}
}
