package main

import (
	config "inngest.com/defs/config"
)

// This config is a template for Terraform.
//
// We use Terraform templating to inject service names, hosts, and secrets
// when provisioning and deploying our infrastructure.
config.#Config & {
	log: {
		format: "json"
		level:  "info"
	}

	eventAPI: {
		addr: "0.0.0.0"
		port: 80
	}

	execution: {
		drivers: {
			docker: config.#DockerDriver
			http:   config.#HTTPDriver
		}
		logOutput: true
	}

	state: {
		service: config.#RedisState & {
			host: "${REDIS_HOST}"
			port: 6379
		}
	}

	eventstream: {
		service: config.#SQSMessaging & {
			queueURL: "${EVENT_SQS_URL}"
			region:   "us-east-2"
			topic:    "events"
		}
	}

	queue: {
		service: config.#SQSQueue & {
			queueURL: "${QUEUE_SQS_URL}"
			region:   "us-east-2"
			topic:    "steps"
		}
	}
}
