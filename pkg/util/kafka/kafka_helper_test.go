package kafka

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	// KafkaDefaultImage is the default Kafka image for tests (KRaft mode)
	KafkaDefaultImage = "confluentinc/cp-kafka:7.7.1"

	// Base port for external listeners (each broker adds nodeID to this)
	baseExternalPort = 29092
)

// KafkaCluster represents a multi-broker Kafka cluster for testing
type KafkaCluster struct {
	containers []testcontainers.Container
	network    *testcontainers.DockerNetwork
	brokers    []string // External broker addresses (host:port)
}

// KafkaOption represents a configuration option for the Kafka cluster
type KafkaOption func(*kafkaConfig)

// kafkaConfig holds the configuration for starting a Kafka cluster
type kafkaConfig struct {
	image      string
	numBrokers int
}

// WithKafkaImage sets a custom Docker image for Kafka
func WithKafkaImage(image string) KafkaOption {
	return func(kc *kafkaConfig) {
		kc.image = image
	}
}

// WithNumBrokers sets the number of brokers in the cluster
func WithNumBrokers(num int) KafkaOption {
	return func(kc *kafkaConfig) {
		kc.numBrokers = num
	}
}

// StartKafkaCluster starts a Kafka cluster with the specified number of brokers
func StartKafkaCluster(t *testing.T, opts ...KafkaOption) (*KafkaCluster, error) {
	config := &kafkaConfig{
		image:      KafkaDefaultImage,
		numBrokers: 3,
	}
	for _, opt := range opts {
		opt(config)
	}

	ctx := t.Context()

	// Create a shared Docker network for broker communication
	net, err := network.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	cluster := &KafkaCluster{
		containers: make([]testcontainers.Container, 0, config.numBrokers),
		network:    net,
		brokers:    make([]string, 0, config.numBrokers),
	}

	// Build the controller quorum voters string
	// Format: "1@kafka-1:9093,2@kafka-2:9093,3@kafka-3:9093"
	var quorumVoters string
	for i := 1; i <= config.numBrokers; i++ {
		if i > 1 {
			quorumVoters += ","
		}
		quorumVoters += fmt.Sprintf("%d@kafka-%d:9093", i, i)
	}

	// Start all brokers - they need to start together for KRaft quorum
	for i := 1; i <= config.numBrokers; i++ {
		brokerContainer, externalAddr, err := startKafkaBroker(ctx, t, config.image, net, i, config.numBrokers, quorumVoters)
		if err != nil {
			// Clean up any containers that were started
			require.NoError(t, cluster.Terminate(ctx))
			return nil, fmt.Errorf("failed to start broker %d: %w", i, err)
		}
		cluster.containers = append(cluster.containers, brokerContainer)
		cluster.brokers = append(cluster.brokers, externalAddr)
	}

	// Wait for cluster to be ready
	if err := cluster.waitForCluster(ctx, t); err != nil {
		require.NoError(t, cluster.Terminate(ctx))
		return nil, fmt.Errorf("cluster failed to become ready: %w", err)
	}

	return cluster, nil
}

// startKafkaBroker starts a single Kafka broker container
func startKafkaBroker(ctx context.Context, t *testing.T, image string, net *testcontainers.DockerNetwork, nodeID, numBrokers int, quorumVoters string) (testcontainers.Container, string, error) {
	containerName := fmt.Sprintf("kafka-%d", nodeID)
	internalPort := 9092
	controllerPort := 9093
	// Each broker gets a unique external port: 29092 + nodeID (so 29093, 29094, 29095)
	externalPort := baseExternalPort + nodeID

	// All nodes are both controller and broker in KRaft mode
	processRoles := "broker,controller"

	env := map[string]string{
		// KRaft mode settings
		"KAFKA_NODE_ID":                    strconv.Itoa(nodeID),
		"KAFKA_PROCESS_ROLES":              processRoles,
		"KAFKA_CONTROLLER_QUORUM_VOTERS":   quorumVoters,
		"CLUSTER_ID":                       "MkU3OEVBNTcwNTJENDM2Qk", // Fixed cluster ID
		"KAFKA_CONTROLLER_LISTENER_NAMES":  "CONTROLLER",
		"KAFKA_INTER_BROKER_LISTENER_NAME": "INTERNAL",

		// Listener configuration with fixed external port
		"KAFKA_LISTENERS": fmt.Sprintf(
			"INTERNAL://0.0.0.0:%d,EXTERNAL://0.0.0.0:%d,CONTROLLER://0.0.0.0:%d",
			internalPort, externalPort, controllerPort,
		),
		"KAFKA_ADVERTISED_LISTENERS": fmt.Sprintf(
			"INTERNAL://%s:%d,EXTERNAL://localhost:%d",
			containerName, internalPort, externalPort,
		),
		"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP": "INTERNAL:PLAINTEXT,EXTERNAL:PLAINTEXT,CONTROLLER:PLAINTEXT",

		// Replication settings
		"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR":         strconv.Itoa(numBrokers),
		"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR": strconv.Itoa(numBrokers),
		"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR":            "1",
		"KAFKA_DEFAULT_REPLICATION_FACTOR":               strconv.Itoa(numBrokers),
		"KAFKA_MIN_INSYNC_REPLICAS":                      "1",

		// Performance settings for testing
		"KAFKA_NUM_PARTITIONS":            "1",
		"KAFKA_LOG_RETENTION_HOURS":       "1",
		"KAFKA_LOG_SEGMENT_BYTES":         "1073741824",
		"KAFKA_AUTO_CREATE_TOPICS_ENABLE": "false",
	}

	externalPortNat := nat.Port(fmt.Sprintf("%d/tcp", externalPort))

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{string(externalPortNat)},
		Networks:     []string{net.Name},
		NetworkAliases: map[string][]string{
			net.Name: {containerName},
		},
		Env: env,
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Memory = 1024 * 1024 * 1024 // 1GB
			// Bind to fixed host port
			hc.PortBindings = nat.PortMap{
				externalPortNat: []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: strconv.Itoa(externalPort)},
				},
			}
		},
		// Don't wait for full startup - KRaft needs quorum first
		// Just wait for the process to start
		WaitingFor: wait.ForLog("Starting controller").WithStartupTimeout(60 * time.Second),
	}

	genericContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create container: %w", err)
	}

	externalAddr := fmt.Sprintf("localhost:%d", externalPort)
	t.Logf("Kafka broker %d started, external address: %s", nodeID, externalAddr)

	return genericContainer, externalAddr, nil
}

// waitForCluster waits for the cluster to be fully operational
func (k *KafkaCluster) waitForCluster(ctx context.Context, t *testing.T) error {
	// Create a client to verify the cluster
	client, err := kgo.NewClient(
		kgo.SeedBrokers(k.brokers...),
		kgo.RetryBackoffFn(func(attempt int) time.Duration {
			return time.Second
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Wait for all brokers to be visible
	deadline := time.Now().Add(120 * time.Second)
	admin := kadm.NewClient(client)
	for time.Now().Before(deadline) {
		meta, err := admin.BrokerMetadata(ctx)
		if err == nil && len(meta.Brokers) == len(k.containers) {
			t.Logf("Kafka cluster ready with %d brokers", len(meta.Brokers))
			return nil
		}
		if err != nil {
			t.Logf("Waiting for cluster: %v", err)
		} else {
			t.Logf("Waiting for cluster: have %d/%d brokers", len(meta.Brokers), len(k.containers))
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("timeout waiting for all brokers to join cluster")
}

// Brokers returns the bootstrap server addresses
func (k *KafkaCluster) Brokers() []string {
	return k.brokers
}

// BrokersString returns the bootstrap servers as a comma-separated string
func (k *KafkaCluster) BrokersString() string {
	return strings.Join(k.brokers, ",")
}

// CreateTopic creates a topic with the specified configuration
func (k *KafkaCluster) CreateTopic(ctx context.Context, name string, partitions int32, replicationFactor int16, minISR int) error {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(k.brokers...),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	admin := kadm.NewClient(client)

	// Create topic with configs
	configs := map[string]*string{
		"min.insync.replicas": stringPtr(strconv.Itoa(minISR)),
	}

	resp, err := admin.CreateTopics(ctx, partitions, replicationFactor, configs, name)
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	for _, topic := range resp {
		if topic.Err != nil {
			return fmt.Errorf("failed to create topic %s: %w", topic.Topic, topic.Err)
		}
	}

	return nil
}

func stringPtr(s string) *string {
	return &s
}

// Terminate stops and removes all containers and the network
func (k *KafkaCluster) Terminate(ctx context.Context) error {
	var errs []error

	for _, c := range k.containers {
		if c != nil {
			if err := c.Terminate(ctx); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if k.network != nil {
		if err := k.network.Remove(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during termination: %v", errs)
	}

	return nil
}

// NewClient creates a new Kafka client connected to the cluster
func (k *KafkaCluster) NewClient(opts ...kgo.Opt) (*kgo.Client, error) {
	allOpts := append([]kgo.Opt{kgo.SeedBrokers(k.brokers...)}, opts...)
	return kgo.NewClient(allOpts...)
}

// NewAdminClient creates a new Kafka admin client connected to the cluster
func (k *KafkaCluster) NewAdminClient() (*kadm.Client, error) {
	client, err := k.NewClient()
	if err != nil {
		return nil, err
	}
	return kadm.NewClient(client), nil
}

// StopBroker stops a specific broker by index (0-based)
func (k *KafkaCluster) StopBroker(ctx context.Context, index int) error {
	if index < 0 || index >= len(k.containers) {
		return fmt.Errorf("broker index %d out of range (0-%d)", index, len(k.containers)-1)
	}

	if err := k.containers[index].Stop(ctx, nil); err != nil {
		return fmt.Errorf("failed to stop broker %d: %w", index, err)
	}

	return nil
}

// GetPartitionLeader returns the broker index (0-based) that is leader for the given topic/partition.
// It retries for up to 30 seconds waiting for the partition metadata to be available.
func (k *KafkaCluster) GetPartitionLeader(ctx context.Context, topic string, partition int32) (int, error) {
	client, err := k.NewClient()
	if err != nil {
		return -1, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	admin := kadm.NewClient(client)

	// Retry for up to 30 seconds waiting for partition metadata to be available
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		metadata, err := admin.Metadata(ctx, topic)
		if err != nil {
			lastErr = fmt.Errorf("failed to get metadata: %w", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		topicMeta, ok := metadata.Topics[topic]
		if !ok {
			lastErr = fmt.Errorf("topic %s not found", topic)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, p := range topicMeta.Partitions {
			if p.Partition == partition {
				// Broker IDs in our cluster are 1-indexed (nodeID starts at 1)
				// Convert to 0-indexed for container array access
				return int(p.Leader) - 1, nil
			}
		}

		lastErr = fmt.Errorf("partition %d not found for topic %s", partition, topic)
		time.Sleep(500 * time.Millisecond)
	}

	return -1, lastErr
}

// StartKafkaClusterWithTopic is a convenience function that starts a cluster and creates a topic
func StartKafkaClusterWithTopic(t *testing.T, topicName string, partitions int32, replicationFactor int16, minISR int, opts ...KafkaOption) (*KafkaCluster, error) {
	cluster, err := StartKafkaCluster(t, opts...)
	if err != nil {
		return nil, err
	}

	ctx := t.Context()
	if err := cluster.CreateTopic(ctx, topicName, partitions, replicationFactor, minISR); err != nil {
		require.NoError(t, cluster.Terminate(ctx))
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}

	t.Logf("Created topic %s with partitions=%d, replication=%d, minISR=%d", topicName, partitions, replicationFactor, minISR)

	return cluster, nil
}
