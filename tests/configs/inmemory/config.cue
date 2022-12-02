package tests

import (
	config "inngest.com/defs/config"
)

// In-memory everything.
config.#Config & {
	log: {
		format: "json"
		level:  "trace"
	}

	eventAPI: addr: "127.0.0.1"

	execution: {
		logOutput: true
		drivers: {
			docker: config.#DockerDriver
			http: config.#DockerDriver
		}
	}
}
