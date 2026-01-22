package testutil

// Container image versions for testing
const (
	// ValkeyDefaultImage is the default Valkey image version for tests
	ValkeyDefaultImage = "valkey/valkey:8.0.1"

	// ValkeyDefaultImageAlpine is the default Valkey image version for tests
	ValkeyDefaultImageAlpine = "docker.io/valkey/valkey:8.0.1-alpine"

	// GarnetDefaultImage is the default Garnet image version for tests
	GarnetDefaultImage = "ghcr.io/microsoft/garnet:1.0.87"

	// KafkaDefaultImage is the default Kafka image for tests (KRaft mode)
	KafkaDefaultImage = "confluentinc/cp-kafka:7.7.1"
)
