package config

// Config defines the top-level config for all services.
#Config: {
	log: {
		level:  ("trace" | "debug" | "info" | "warn") | *"info"
		format: "json" | *"json"
	}

	// EventAPI is used to configure the API for listening to events.
	eventAPI: {
		addr: string | *"0.0.0.0"
		port: string | *"8288"
	}

	// eventStream is used to configure the event stream pub/sub implementation.  This
	// pub-sub stream is used to send and receive events between the event API and the
	// executors which initialize and work with events.
	eventStream: {
		// Default to an in-memory pubsub using the "events" topic.
		service: #MessagingService | *{backend: "inmemory", topic: "events"}
	}
}

// @TODO: "inmemory" | "aws-sqs" | "gcp-pubsub" | "nats" | "redis" | *"inmemory"
#MessagingService: #InmemMessaging | #NATSMessaging

// InmemMessaging defines configuration for an in-memory based event queue.  This is
// only usable for single-container testing;  in-memory implementations only share
// events within the same process.
#InmemMessaging: {
	backend: "inmemory"
	topic:   string
}

// NATSMessaging defines configuration for using NATS as a backing event stream.
#NATSMessaging: {
	backend:   "nats"
	topic:     string
	serverURL: string
}
