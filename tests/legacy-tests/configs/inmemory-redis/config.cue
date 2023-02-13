package tests

import (
	config "inngest.com/defs/config"
)

// This configuration uses SQS for the backing event stream _and_ the
// backing queue for organizing and running steps.  This lets us assert
// that the SQS implementatinos work as expected.
config.#Config & {
	log: {
		format: "json"
		level:  "trace"
	}

	eventAPI: {
		addr: "127.0.0.1"
	}

	state: {
		service: config.#RedisState & {
			host: "127.0.0.1"
			port: 6379
		}
	}

	execution: {
		drivers: {
			docker: config.#DockerDriver
			http:   config.#HTTPDriver
		}
		logOutput: true
	}
}
