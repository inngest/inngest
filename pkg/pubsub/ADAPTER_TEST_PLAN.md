# Pub/Sub Adapter — Test Plans

## Testing Strategy

### Conformance Test Suite

Every adapter must pass a shared **conformance test suite** that validates the `Adapter` interface
contract. This ensures behavioral consistency across backends while allowing backend-specific
tests for unique features.

```go
// pkg/pubsub/adaptertest/conformance.go

// RunConformance runs the full adapter conformance suite against the provided adapter.
// Each adapter's _test.go calls this with its own setup/teardown.
func RunConformance(t *testing.T, setup func(t *testing.T) Adapter) {
    t.Run("Publish", func(t *testing.T) { ... })
    t.Run("Subscribe", func(t *testing.T) { ... })
    t.Run("Concurrency", func(t *testing.T) { ... })
    t.Run("Acknowledgment", func(t *testing.T) { ... })
    t.Run("Lifecycle", func(t *testing.T) { ... })
}
```

Usage in each adapter:
```go
// pkg/pubsub/adapters/kafka/kafka_test.go
func TestKafkaConformance(t *testing.T) {
    adaptertest.RunConformance(t, func(t *testing.T) pubsub.Adapter {
        return newTestKafkaAdapter(t)
    })
}
```

---

## Conformance Tests (all adapters must pass)

### 1. Publish / Subscribe Basics

| Test | Description |
|------|-------------|
| `TestPublishAndReceive` | Publish a message, subscribe, assert received message matches sent (Name, Version, Data, Metadata). |
| `TestPublishMultipleMessages` | Publish N messages, subscribe, assert all N received (order not guaranteed unless backend guarantees it). |
| `TestSubscribeBlocksUntilCancel` | Subscribe in a goroutine, cancel context, assert Subscribe returns nil error. |
| `TestPublishNoSubscriber` | Publish with no active subscriber — must not error (fire-and-forget semantics). |
| `TestSubscribeReceivesOnlyAfterStart` | Publish before subscribe starts, then subscribe — behavior is backend-defined but must not panic or hang. |
| `TestMessageDataIntegrity` | Publish messages with varying data sizes (empty, 1 byte, 1KB, 1MB) — assert exact byte-level match on receive. |

### 2. Concurrency

| Test | Description |
|------|-------------|
| `TestConcurrency` | Subscribe with `WithConcurrency(10)`, publish 10 messages that each block until signaled. Assert all 10 handlers are running concurrently. |
| `TestConcurrencyLimit` | Subscribe with `WithConcurrency(5)`, publish 10 blocking messages. Assert only 5 handlers run concurrently; remaining 5 start after first batch completes. |
| `TestConcurrencyOne` | Subscribe with `WithConcurrency(1)` (or default). Publish 3 messages. Assert sequential processing — second handler starts only after first completes. |

### 3. Acknowledgment

| Test | Description |
|------|-------------|
| `TestAckRemovesMessage` | Handler calls `Ack()`. Assert the message is not redelivered within a reasonable timeout. |
| `TestNackRedelivers` | Handler calls `Nack()`. Assert the message is redelivered (for backends that support it). Backends without nack support may skip. |
| `TestRequeueWithDelay` | Handler calls `Requeue(500ms)`. Assert the message is redelivered after approximately 500ms. Backends without delay support should redeliver immediately or skip. |
| `TestNoAckTimesOut` | Handler neither acks nor nacks. Assert the message is eventually redelivered (for backends with ack deadlines). In-memory can skip. |
| `TestDoubleAck` | Handler calls `Ack()` twice. Must not panic. |
| `TestAckAfterNack` | Handler calls `Nack()` then `Ack()`. First call wins. Must not panic. |

### 4. Lifecycle

| Test | Description |
|------|-------------|
| `TestCloseStopsSubscribe` | Call `Close()` while Subscribe is blocking. Assert Subscribe returns without error. |
| `TestCloseWaitsForInflight` | Handler is processing a message. Call `Close()`. Assert Close blocks until handler finishes. |
| `TestCloseThenPublishErrors` | Call `Close()`, then `Publish()`. Assert Publish returns an error. |
| `TestCloseThenSubscribeErrors` | Call `Close()`, then `Subscribe()`. Assert Subscribe returns an error. |
| `TestCloseIdempotent` | Call `Close()` twice. Must not panic or error on second call. |
| `TestHealthyBeforeClose` | Call `Healthy()` on a connected adapter. Assert nil error. |
| `TestHealthyAfterClose` | Call `Close()`, then `Healthy()`. Assert non-nil error. |

### 5. Context Cancellation

| Test | Description |
|------|-------------|
| `TestPublishRespectsContext` | Pass an already-cancelled context to Publish. Assert error. |
| `TestSubscribeCancelDrainsGracefully` | Subscribe with concurrency 5, publish 5 slow messages. Cancel context. Assert all 5 in-flight handlers complete before Subscribe returns. |
| `TestSubscribeContextDeadline` | Subscribe with a context that has a 2s deadline. Assert Subscribe returns after ~2s without error. |

### 6. Subscribe Options

| Test | Description |
|------|-------------|
| `TestDefaultConcurrency` | Subscribe without `WithConcurrency`. Assert messages are processed (default concurrency = 1). |
| `TestUnknownOptionsIgnored` | Future-proof: passing unrecognized options must not panic. |

---

## Backend-Specific Tests

### In-Memory (`adapters/memory/`)

| Test | Description |
|------|-------------|
| `TestFanOut` | Two subscribers on the same topic each receive every published message (fan-out, not competing consumers). |
| `TestIsolatedTopics` | Messages on topic A are not received by subscribers on topic B. |
| `TestNoExternalDependencies` | Adapter creates and operates with zero config (no URLs, no env vars). |

### NATS / JetStream (`adapters/nats/`)

Requires a running NATS server. Use `nats-server -js` in CI via Docker or testcontainers.

| Test | Description |
|------|-------------|
| `TestCoreNATSPublishSubscribe` | Non-JetStream mode: fire-and-forget publish, subscriber receives. |
| `TestJetStreamDurableConsumer` | JetStream mode: publish, stop subscriber, restart subscriber — receives messages published while offline. |
| `TestQueueGroup` | Two subscribers with same `WithConsumerGroup("g")` — each message delivered to exactly one subscriber (competing consumers). |
| `TestQueueGroupDifferentGroups` | Two subscribers with different consumer groups — each receives every message. |
| `TestNATSHeaders` | Publish with `Headers`. Assert headers are present on received message. |
| `TestJetStreamAckNack` | Nack in JetStream mode triggers redelivery. |
| `TestConnectionReconnect` | Disconnect NATS server, reconnect. Assert adapter recovers and Healthy() returns nil after reconnect. |
| `TestDrainOnClose` | Publish async with JetStream. Call Close(). Assert all buffered messages are flushed. |

### Kafka (`adapters/kafka/`) — franz-go

Requires a running Kafka broker. Use `redpanda` or `kafka` container in CI.

| Test | Description |
|------|-------------|
| `TestProduceConsume` | Produce a record, consume it. Assert key, value, headers match. |
| `TestPartitionKey` | Publish two messages with the same `PartitionKey`. Assert both land on the same partition. |
| `TestConsumerGroup` | Two subscribers with same `WithConsumerGroup`. Publish 100 messages. Assert total received across both = 100 (no duplicates). |
| `TestConsumerGroupRebalance` | Start subscriber A in group G. Start subscriber B in same group G. Assert partitions are rebalanced. Stop subscriber B. Assert A takes all partitions back. |
| `TestKafkaHeaders` | Publish with `Headers`. Assert headers round-trip correctly. |
| `TestOffsetEarliest` | Publish 10 messages. Subscribe with `WithStartOffset(OffsetEarliest)`. Assert all 10 received. |
| `TestOffsetLatest` | Publish 10 messages. Subscribe with `WithStartOffset(OffsetLatest)`. Publish 5 more. Assert only 5 received. |
| `TestCommitOnAck` | Ack a message. Restart consumer with same group. Assert acked messages are not re-received. |
| `TestNackDoesNotCommit` | Nack a message. Restart consumer with same group. Assert nacked message is re-received. |
| `TestBatchPerformance` | Publish 10,000 messages. Consume all. Assert throughput is reasonable (no per-message round-trips). |
| `TestTLS` | Connect with TLS config. Assert connection succeeds and Healthy() returns nil. |
| `TestSASL` | Connect with SASL credentials. Assert connection succeeds. |

### AWS SQS (`adapters/sqs/`)

Use localstack in CI for SQS emulation.

| Test | Description |
|------|-------------|
| `TestSQSSendReceive` | Send message, receive it, assert match. |
| `TestSQSDelaySeconds` | Publish with `Delay: 3s`. Assert message is not received for ~3s. |
| `TestSQSFIFOMessageGroup` | Use FIFO queue. Publish with `PartitionKey` (message group ID). Assert FIFO ordering within group. |
| `TestSQSVisibilityTimeout` | Receive message, don't ack. Assert message reappears after visibility timeout. |
| `TestSQSRequeue` | Call `Requeue(5s)`. Assert message reappears after ~5s (via `ChangeMessageVisibility`). |
| `TestSQSDeleteOnAck` | Ack a message. Assert `ReceiveMessage` does not return it again. |
| `TestSQSBatchReceive` | Publish 10 messages. Assert adapter receives them efficiently (up to 10 per `ReceiveMessage` call). |
| `TestSQSLongPolling` | Subscribe to empty queue. Assert no busy-loop (adapter should use `WaitTimeSeconds`). Verify via metric or timing. |
| `TestSNSPublish` | If `TopicARN` configured, publish goes to SNS. Assert SQS subscription receives it. |

### GCP Pub/Sub (`adapters/gcppubsub/`)

Use the GCP Pub/Sub emulator in CI (`gcloud beta emulators pubsub start`).

| Test | Description |
|------|-------------|
| `TestGCPPublishSubscribe` | Publish to topic, receive from subscription. |
| `TestGCPOrderingKey` | Publish with `OrderingKey`. Assert messages with the same ordering key are received in order. |
| `TestGCPAckDeadline` | Subscribe with `WithAckDeadline(5s)`. Don't ack. Assert redelivery after ~5s. |
| `TestGCPMultipleSubscriptions` | Two subscriptions on the same topic (different groups). Assert both receive every message. |
| `TestGCPNack` | Nack a message. Assert it is redelivered. |
| `TestGCPAttributeRoundTrip` | Publish with `Metadata`. Assert attributes arrive on received message. |

### RabbitMQ (`adapters/rabbitmq/`)

Use RabbitMQ container in CI.

| Test | Description |
|------|-------------|
| `TestDirectExchange` | Publish with routing key, bind queue with matching key. Assert received. |
| `TestTopicExchange` | Publish with routing key `foo.bar`. Bind with `foo.*`. Assert received. Bind with `baz.*`. Assert not received. |
| `TestFanoutExchange` | Two queues bound to fanout exchange. Assert both receive every message. |
| `TestHeaderExchange` | Publish with `Headers`. Bind queue with header match. Assert routing works. |
| `TestAckNackRequeue` | Nack with requeue. Assert message redelivered. Nack without requeue (dead-letter). Assert not redelivered. |
| `TestDurableQueue` | Declare durable queue. Publish. Restart subscriber. Assert message still available. |
| `TestPrefetchCount` | Set `WithConcurrency(5)`. Assert RabbitMQ `Qos` prefetch matches. Only 5 unacked messages at a time. |
| `TestConnectionRecovery` | Kill RabbitMQ connection. Assert adapter reconnects automatically. |

---

## Middleware Tests (`middleware/`)

### Logging Middleware

| Test | Description |
|------|-------------|
| `TestLogsOnPublish` | Publish a message. Assert log entry with topic name and message name. |
| `TestLogsOnSubscribeStart` | Subscribe. Assert log entry indicating subscription started. |
| `TestLogsOnError` | Publish to a closed adapter. Assert error is logged. |

### Metrics Middleware

| Test | Description |
|------|-------------|
| `TestPublishCounter` | Publish 5 messages. Assert `pubsub_publish_total` counter = 5. |
| `TestPublishErrorCounter` | Publish to closed adapter. Assert `pubsub_publish_errors_total` incremented. |
| `TestSubscribeLatencyHistogram` | Subscribe and process a message. Assert `pubsub_subscribe_duration_seconds` has an observation. |
| `TestMetricsHaveTopicLabel` | Publish to two topics. Assert metrics are labeled by topic. |

### Tracing Middleware

| Test | Description |
|------|-------------|
| `TestPublishCreatesSpan` | Publish a message. Assert a span with operation `pubsub.publish` and topic attribute. |
| `TestSubscribeCreatesSpan` | Receive a message. Assert a span with operation `pubsub.process`. |
| `TestTraceContextPropagation` | Publish with active trace context. Assert subscriber's span has the same trace ID (context propagated via headers). |

---

## Concurrency Helper Tests (`concurrency.go`)

| Test | Description |
|------|-------------|
| `TestConcurrentSubscriberRespectsConcurrency` | Set concurrency=3. Feed 10 messages via receive func. Assert max 3 handlers running at once. |
| `TestConcurrentSubscriberCancelDrains` | Start with concurrency=5, feed 5 slow messages. Cancel context. Assert function returns only after all 5 complete. |
| `TestConcurrentSubscriberReceiveError` | Receive func returns error. Assert `ConcurrentSubscriber` returns that error after draining in-flight work. |

---

## Registry Tests (`registry.go`)

| Test | Description |
|------|-------------|
| `TestRegisterAndCreate` | Register a factory, call `NewAdapter`. Assert adapter is returned. |
| `TestUnknownBackendErrors` | Call `NewAdapter` with unregistered name. Assert descriptive error. |
| `TestRegisterOverwrites` | Register same name twice. Assert second factory wins. |
| `TestConcurrentRegisterSafe` | Register from multiple goroutines. Assert no race (run with `-race`). |

---

## Integration / End-to-End Tests

These tests run against real backends (Docker containers in CI) and test the full
publish → subscribe → ack cycle.

| Test | Description |
|------|-------------|
| `TestE2E_MemoryAdapter` | Full cycle with in-memory adapter. Runs in every CI build. |
| `TestE2E_NATSAdapter` | Full cycle with NATS container. |
| `TestE2E_KafkaAdapter` | Full cycle with Redpanda/Kafka container. |
| `TestE2E_SQSAdapter` | Full cycle with LocalStack container. |
| `TestE2E_GCPPubSubAdapter` | Full cycle with GCP emulator container. |
| `TestE2E_RabbitMQAdapter` | Full cycle with RabbitMQ container. |

Each E2E test:
1. Starts the backend container (via testcontainers-go or CI service).
2. Creates the adapter.
3. Asserts `Healthy()` returns nil.
4. Publishes 100 messages.
5. Subscribes with concurrency 10.
6. Asserts all 100 received and acked.
7. Calls `Close()`.
8. Asserts graceful shutdown (no lost messages).

---

## CI Configuration

```yaml
# .github/workflows/pubsub-tests.yml (sketch)
services:
  nats:
    image: nats:latest
    command: -js
    ports: [4222:4222]
  redpanda:
    image: redpandadata/redpanda:latest
    ports: [9092:9092]
  localstack:
    image: localstack/localstack:latest
    ports: [4566:4566]
  pubsub-emulator:
    image: gcr.io/google.com/cloudsdktool/google-cloud-cli:emulators
    ports: [8085:8085]
  rabbitmq:
    image: rabbitmq:3-management
    ports: [5672:5672]
```

Backend-specific tests use build tags to skip when the backend is unavailable:

```go
//go:build integration && kafka

package kafka_test
```

Run: `go test -tags=integration,kafka ./pkg/pubsub/adapters/kafka/`

## Test File Layout

```
pkg/pubsub/
    adaptertest/
        conformance.go          # Shared conformance suite
    concurrency_test.go         # ConcurrentSubscriber helper tests
    registry_test.go            # Registry tests
    middleware/
        logging_test.go
        metrics_test.go
        tracing_test.go
    adapters/
        memory/
            memory_test.go      # Conformance + memory-specific
        nats/
            nats_test.go        # Conformance + NATS-specific (build tag: integration,nats)
        kafka/
            kafka_test.go       # Conformance + Kafka-specific (build tag: integration,kafka)
        sqs/
            sqs_test.go         # Conformance + SQS-specific (build tag: integration,sqs)
        gcppubsub/
            gcppubsub_test.go   # Conformance + GCP-specific (build tag: integration,gcppubsub)
        rabbitmq/
            rabbitmq_test.go    # Conformance + RabbitMQ-specific (build tag: integration,rabbitmq)
```
