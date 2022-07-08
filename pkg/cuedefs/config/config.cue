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

	// eventstream is used to configure the event stream pub/sub implementation.  This
	// pub-sub stream is used to send and receive events between the event API and the
	// executors which initialize and work with events.
	eventstream: {
		// Default to an in-memory pubsub using the "events" topic.
		service: #MessagingService | *{backend: "inmemory", topic: "events"}
		// This struct is retained for any shared settings
	}

	queue: {
		// @TODO: Add SQS.
		service: #QueueService | *{backend: "inmemory"}
		// This struct is retained for any shared settings
	}

	state: {
		service: #StateService | *{backend: "inmemory"}
		// This struct is retained for any shared settings
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

#QueueService: #InmemQueue

#InmemQueue: {
	backend: "inmemory"
}

#StateService: #InmemState | #RedisState

#InmemState: {
	backend: "inmemory"
}

#RedisState: {
	backend: "redis"

	host:        string | *"localhost"
	port:        >0 & <=65535 | *6379
	db:          >=0 | *0
	username?:   string
	password?:   string
	maxRetries?: >=-1 | *3
	poolSize?:   >=1
}
