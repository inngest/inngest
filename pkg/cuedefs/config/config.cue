package config

// Config defines the top-level config for all services.
#Config: {
	log: {
		level:  ("trace" | "debug" | "info" | "warn" | "error") | *"info"
		format: "json" | "console" | *"json"
	}

	// EventAPI is used to configure the API for listening to events.
	eventAPI: {
		addr: string | *"0.0.0.0"
		port: >0 & <=65535 | *8288

		// maxSize represents the maximum size of events read by the
		// event API.  Any events over this size limit will be rejected
		// with an HTTP 413 (Request Entity Too Large).
		//
		// NOTE: Some event stream implementations have their own limits
		// (eg. SQS is 256kb).
		maxSize: >=1024 | *(512 * 1024)
	}

	// CoreAPI is used to configure the API for manging the system
	coreAPI: {
		addr: string | *"0.0.0.0"
		port: >0 & <=65535 | *8300
	}

	execution: {
		// Enable drivers for given runtimes within this array.  The key
		// is the runtime name specified within steps of a function, and
		// the value is the specific driver to use for these runtimes.
		//
		// This allows you to build and specify alternate drivers for each
		// runtime, eg. a Kubernetes driver for the docker runtime.
		//
		// By default, enable the docker and HTTP drivers for the docker and
		// HTTP runtimes respectively.
		drivers: {
			// For each runtime, specify a driver which has its own name.
			[runtime=_]: #Driver & {name: string}
		} | *{
			http: #HTTPDriver
			connect: #ConnectDriver
		}

		// logOutput logs output from steps within logs.  This may
		// result in large logs and sensitive data being printed
		// to stderr, and is only intended for development.
		logOutput: bool | *false
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
		service: #QueueService | *{backend: "redis"}
		// This struct is retained for any shared settings
	}

	state: {
		service: #StateService | *{backend: "redis"}
		// This struct is retained for any shared settings
	}

	datastore: {
		service: #DataStoreService | *{backend: "inmemory"}
		// This struct is retained for any shared settings
	}
}

// @TODO: Add custom redis driver, add Kafka.
#MessagingService: #InmemMessaging | #NATSMessaging | #SQSMessaging | #GCPPubSubMessaging

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

// GCPPubSubMessaging defines configuration for using GCP's Pub/Sub as a
// backing event stream.
#GCPPubSubMessaging: {
	backend: "gcp-pubsub"
	project: string
	topic:   string
}

// SQSMessaging defines configuration for using SQS as a backing event stream.
#SQSMessaging: {
	backend:  "aws-sqs"
	queueURL: string
	region:   string
	topic:    string
}

// # Queues
//

#QueueService: #InmemQueue | #SQSQueue | #RedisQueue

#InmemQueue: {
	// This uses the Redis driver with an in-memory redis instance
	// under the hood.
	backend: "inmemory"
}

#SQSQueue: {
	backend: "aws-sqs"

	queueURL: string
	region:   string
	// Topic stores the topic for all queue-related items.  This must be its
	// own unique topic for processing enqueued steps.
	topic: string

	// concurrency specifies how many concurrent queue items - and therefore
	// function steps - can be handled in parallel.
	concurrency: >=1 | *10
}

// RedisState uses Redis as the backend state store.
#RedisQueue: {
	backend: "redis"

	// If DSN is supplied (eg. redis://user:pass@host:port/db), this
	// will override any of the options provided below.
	dsn?: string

	host:        string | *"localhost"
	port:        >0 & <=65535 | *6379
	db:          >=0 | *0
	username?:   string
	password?:   string
	maxRetries?: >=-1 | *3
	poolSize?:   >=1

	// keyPrefix is the prefix used for all redis keys stored
	keyPrefix: string | *""
}

// # State
//
// State stores distributed state when running functions.  You can choose one of
// StateServices as the backend to host state.
#StateService: #InmemState | #RedisState

// InmemState stores state in memory, local to each process.  This should only
// be used for development or testing, but never for production.
#InmemState: {
	backend: "inmemory"
}

// RedisState uses Redis as the backend state store.
#RedisState: {
	backend: "redis"

	// If DSN is supplied (eg. redis://user:pass@host:port/db), this
	// will override any of the options provided below.
	dsn?: string

	host:        string | *"localhost"
	port:        >0 & <=65535 | *6379
	db:          >=0 | *0
	username?:   string
	password?:   string
	maxRetries?: >=-1 | *3
	poolSize?:   >=1

	// keyPrefix is the prefix used for all redis keys stored
	keyPrefix: string | *"inngest:state"
}

// # DataStore
//
// DataStore stores the persisted system data including Functions and Actions versions
#DataStoreService: #InmemDataStore | #PostgresDataStore

// InmemDataStore stores data in memory, local to each process. This should only
// be used for development or testing, never for production.
#InmemDataStore: {
	backend: "inmemory"
}

// PostgresDataStore uses PostgreSQL
#PostgresDataStore: {
	backend: "postgres"
	URI:     string | *"postgres://localhost:5432/postgres?sslmode=disable"
}

// Drivers handle execution of each step within a function.
#Driver: #MockDriver | #HTTPDriver | #ConnectDriver

// MockDriver is used in testing to mock and stub function executions.  You
// almost certainly do not need to include this in your config.
#MockDriver: {
	name:    "mock"
	driver?: string | *"mock"
}

#HTTPDriver: {
	name:        "http"
	timeout?:    int | *7200 // 2 hours
	signingKey?: string
}

#ConnectDriver: {
	name:        "connect"
}
