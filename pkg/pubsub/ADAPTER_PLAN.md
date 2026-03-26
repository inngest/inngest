# Pub/Sub Adapter Pattern — Implementation Plan

## Problem Statement

The current pub/sub implementation is tightly coupled to `gocloud.dev/pubsub` via URL-based
routing (`URLMux`). This creates several limitations:

1. **Every backend must have a gocloud.dev driver** — rules out Kafka entirely, limits advanced
   features of NATS JetStream, SQS, and RabbitMQ.
2. **`NatsConnector` already bypasses the abstraction** — it implements its own publish/subscribe
   with JetStream but doesn't satisfy the `Publisher`/`Subscriber` interfaces, proving the
   current abstraction is insufficient.
3. **`TopicURLCreator` is a URL formatter, not an adapter** — it can't express consumer groups,
   partition keys, routing keys, visibility timeouts, or ordering keys.
4. **No lifecycle management** — no `Close()` or `Healthy()` on the core interfaces.
5. **Hard-coded backend `switch`** in `config/messaging.go` — adding a backend means editing
   shared code.
6. **Concurrency, ack/nack, and retry logic are baked into the single `broker` struct** — not
   reusable or overridable per backend.

## Goals

- Define a clean `Adapter` interface that each backend implements directly.
- Support Kafka (via **franz-go**), NATS/JetStream, AWS SQS, GCP Pub/Sub, RabbitMQ, and
  in-memory (testing) without lowest-common-denominator compromises.
- Give each adapter full control over connections, consumer groups, ack/nack, and concurrency.
- Make the system extensible via a self-registering factory (no hard-coded switch).
- Enable cross-cutting concerns (metrics, tracing, retry) via middleware decorators.
- **Full backward compatibility** — existing callers (`Publisher`, `Subscriber`,
  `PublishSubscriber`, `PerformFunc`, `Message`) must continue to compile and work unchanged
  throughout the entire migration. No big-bang cutover.

## Backward Compatibility Contract

The following public API surface **must not break** at any point during this project:

```go
// Existing interfaces — preserved as-is
type Publisher interface {
    Publish(ctx context.Context, topic string, m Message) error
}
type Subscriber interface {
    Subscribe(ctx context.Context, topic string, handler PerformFunc) error
    SubscribeN(ctx context.Context, topic string, handler PerformFunc, concurrency int64) error
}
type PublishSubscriber interface {
    Publisher
    Subscriber
}

// Existing constructors — preserved as-is
func NewPublisher(ctx context.Context, c config.MessagingService) (Publisher, error)
func NewSubscriber(ctx context.Context, c config.MessagingService) (Subscriber, error)
func NewPublishSubscriber(ctx context.Context, c config.MessagingService) (PublishSubscriber, error)

// Existing types — preserved as-is
type Message struct { ... }
type PerformFunc func(context.Context, Message) error
```

### Existing callers (must remain untouched until explicit migration PR):

| Caller | Usage |
|--------|-------|
| `pkg/api/service.go` | `pubsub.NewPublisher`, `pubsub.Publisher`, `pubsub.Message` |
| `pkg/execution/runner/runner.go` | `pubsub.NewPublishSubscriber`, `pubsub.PublishSubscriber`, `pubsub.Publisher`, `pubsub.Message`, `handleMessage(ctx, pubsub.Message)` |
| `pkg/execution/executor/service.go` | `pubsub.NewPublisher`, `pubsub.Publisher`, `pubsub.Message` |
| `pkg/devserver/devserver.go` | `pubsub.NewPublisher`, `pubsub.Publisher` |
| `pkg/devserver/service.go` | `pubsub.Publisher`, `pubsub.Message` |
| `pkg/devserver/lifecycle.go` | `pubsub.Publisher` |
| `pkg/config/messaging.go` | `MessagingService`, `TopicURLCreator`, all backend config structs |

### Strategy: Bridge, don't break

- The old interfaces (`Publisher`, `Subscriber`, `PublishSubscriber`) stay in `pubsub.go`
  permanently — they are the stable API.
- A `LegacyBridge` wraps new `Adapter` implementations to satisfy old interfaces.
- `NewPublisher`/`NewSubscriber`/`NewPublishSubscriber` are updated internally to create
  adapters + bridge, but their signatures and behavior are identical.
- Old `broker.go` + gocloud.dev code is only removed in a final cleanup PR after all callers
  have been verified.

## Design

### Core Adapter Interface (`pkg/pubsub/adapter.go`)

```go
// Adapter is the primary interface every pub/sub backend implements.
type Adapter interface {
    // Publish sends a message to the named topic.
    Publish(ctx context.Context, topic string, msg Message) error

    // Subscribe consumes messages from a topic, calling handler for each.
    // It blocks until ctx is cancelled. Options control concurrency,
    // consumer groups, etc.
    Subscribe(ctx context.Context, topic string, handler HandlerFunc, opts ...SubscribeOption) error

    // Close gracefully shuts down the adapter, flushing in-flight messages.
    Close(ctx context.Context) error

    // Healthy returns nil if the adapter is connected and operational.
    Healthy(ctx context.Context) error
}
```

### New Handler and Ack (`pkg/pubsub/handler.go`)

```go
// HandlerFunc processes a received message. The Acknowledger gives the handler
// explicit control over ack/nack/requeue.
type HandlerFunc func(ctx context.Context, msg Message, ack Acknowledger)

// Acknowledger provides explicit message acknowledgment control.
type Acknowledger interface {
    Ack()
    Nack()
    // Requeue returns the message to the queue with an optional delay.
    // Backends that don't support delay will ignore it.
    Requeue(delay time.Duration)
}
```

### Subscribe Options (`pkg/pubsub/options.go`)

```go
type SubscribeConfig struct {
    Concurrency   int
    ConsumerGroup string        // Kafka consumer group, NATS queue group, SQS queue
    AckDeadline   time.Duration // How long before unacked messages are redelivered
    MaxRetries    int           // Backend-level retry count (0 = backend default)
    StartOffset   StartOffset   // Kafka: earliest/latest; NATS: deliver new/all
}

type StartOffset int
const (
    OffsetDefault StartOffset = iota
    OffsetEarliest
    OffsetLatest
)

type SubscribeOption func(*SubscribeConfig)

func WithConcurrency(n int) SubscribeOption { ... }
func WithConsumerGroup(group string) SubscribeOption { ... }
func WithAckDeadline(d time.Duration) SubscribeOption { ... }
func WithMaxRetries(n int) SubscribeOption { ... }
func WithStartOffset(o StartOffset) SubscribeOption { ... }
```

### Message Type Changes

The existing `Message` struct is preserved. New backend-hint fields are added with `json:"-"`
tags so they don't affect serialization. The existing `Data string` field is kept for backward
compat; a future PR can add a `RawData []byte` field or change encoding.

```go
type Message struct {
    // Existing fields — unchanged
    Name      string         `json:"name"`
    Version   int            `json:"v"`
    Data      string         `json:"data"`
    Timestamp time.Time      `json:"ts"`
    Metadata  map[string]any `json:"meta,omitempty"`

    // New: unique message identifier
    ID string `json:"id,omitempty"`

    // New: backend hints — adapters use what they support, ignore the rest.
    // Not serialized; set in-process before Publish().
    PartitionKey string            `json:"-"` // Kafka partition, SQS message group ID
    OrderingKey  string            `json:"-"` // GCP Pub/Sub ordering key
    RoutingKey   string            `json:"-"` // RabbitMQ routing key
    Delay        time.Duration     `json:"-"` // SQS delay, RabbitMQ delayed message
    Headers      map[string]string `json:"-"` // Kafka headers, RabbitMQ headers, NATS headers
}
```

### Adapter Registry (`pkg/pubsub/registry.go`)

```go
type AdapterFactory func(ctx context.Context, cfg json.RawMessage) (Adapter, error)

var (
    mu       sync.RWMutex
    registry = map[string]AdapterFactory{}
)

func Register(name string, factory AdapterFactory) { ... }
func NewAdapter(ctx context.Context, backend string, cfg json.RawMessage) (Adapter, error) { ... }
```

### Legacy Bridge (`pkg/pubsub/bridge.go`)

```go
// LegacyBridge wraps an Adapter to satisfy the existing Publisher/Subscriber/
// PublishSubscriber interfaces. This allows existing callers to use new adapters
// without code changes.
type LegacyBridge struct {
    adapter Adapter
}

func NewLegacyBridge(a Adapter) *LegacyBridge { ... }

// Publish satisfies Publisher — delegates directly.
func (b *LegacyBridge) Publish(ctx context.Context, topic string, m Message) error { ... }

// Subscribe satisfies Subscriber — wraps old PerformFunc into new HandlerFunc,
// translating error return to Ack/Nack.
func (b *LegacyBridge) Subscribe(ctx context.Context, topic string, handler PerformFunc) error { ... }

// SubscribeN satisfies Subscriber — same wrapping with WithConcurrency(n).
func (b *LegacyBridge) SubscribeN(ctx context.Context, topic string, handler PerformFunc, concurrency int64) error { ... }
```

### Concurrency Helper (`pkg/pubsub/concurrency.go`)

Extract the weighted-semaphore pattern from `broker.go` as a reusable helper:

```go
func ConcurrentSubscriber(
    ctx context.Context,
    concurrency int,
    receive func(ctx context.Context) (Message, Acknowledger, error),
    handler HandlerFunc,
) error { ... }
```

### Middleware (`pkg/pubsub/middleware/`)

```go
type Middleware func(Adapter) Adapter

func Compose(adapter Adapter, mw ...Middleware) Adapter { ... }
```

- `WithLogging` — log publish/subscribe events, errors
- `WithMetrics` — counters, latency histograms
- `WithTracing` — OpenTelemetry span propagation

---

## Milestones

This is a multi-PR project. Each milestone is a separate PR. Within each milestone,
**tests are written first**, then the implementation is added to make them pass.

### PR 1: Core Interfaces + Registry + Conformance Suite

**What**: Foundation that all subsequent adapters build on. No changes to existing code paths.

**Deliverables**:
- `pkg/pubsub/adapter.go` — `Adapter` interface
- `pkg/pubsub/handler.go` — `HandlerFunc`, `Acknowledger`
- `pkg/pubsub/options.go` — `SubscribeConfig`, `SubscribeOption`, option funcs
- `pkg/pubsub/registry.go` — `Register()`, `NewAdapter()`
- `pkg/pubsub/registry_test.go` — registry unit tests
- `pkg/pubsub/concurrency.go` — extracted semaphore helper
- `pkg/pubsub/concurrency_test.go` — concurrency helper tests
- `pkg/pubsub/bridge.go` — `LegacyBridge`
- `pkg/pubsub/bridge_test.go` — bridge tests (verify old interface contract via adapter)
- `pkg/pubsub/adaptertest/conformance.go` — shared conformance test suite
- Add `ID`, `PartitionKey`, `OrderingKey`, `RoutingKey`, `Delay`, `Headers` fields to `Message`

**Backward compat**: No existing files modified (except additive `Message` field additions).
All existing tests pass unchanged.

**Test order**: Write `registry_test.go`, `concurrency_test.go`, `bridge_test.go`, and
`adaptertest/conformance.go` first. Bridge tests use a mock adapter to validate the translation
from `PerformFunc` → `HandlerFunc` and error → ack/nack mapping.

---

### PR 2: Kafka Adapter (franz-go)

**What**: First real adapter. New capability — Kafka was not previously supported.

**Deliverables**:
- `pkg/pubsub/adapters/kafka/kafka.go` — adapter implementation
- `pkg/pubsub/adapters/kafka/kafka_test.go` — conformance + Kafka-specific tests

**Config**:
```go
type Config struct {
    Brokers       []string `json:"brokers"`
    TLS           bool     `json:"tls"`
    SASLMechanism string   `json:"sasl_mechanism,omitempty"`
    SASLUser      string   `json:"sasl_user,omitempty"`
    SASLPass      string   `json:"sasl_pass,omitempty"`
}
```

**Kafka-specific tests** (build tag: `integration,kafka`; CI uses Redpanda container):
- `TestProduceConsume` — basic round-trip
- `TestPartitionKey` — same key lands on same partition
- `TestConsumerGroup` — competing consumers, no duplicates
- `TestConsumerGroupRebalance` — add/remove consumers
- `TestKafkaHeaders` — header round-trip
- `TestOffsetEarliest` / `TestOffsetLatest` — start position
- `TestCommitOnAck` / `TestNackDoesNotCommit` — offset management
- `TestBatchPerformance` — 10k messages, no per-message round-trip
- `TestTLS` / `TestSASL` — auth/encryption

**Backward compat**: Purely additive. No existing files modified.

**Test order**: Write `kafka_test.go` with conformance suite call + all Kafka-specific tests
first (they will fail/skip). Then implement `kafka.go`.

---

### PR 3: In-Memory Adapter

**What**: Replace the gocloud.dev `mem://` backend. Used by all existing unit tests and the
dev server.

**Deliverables**:
- `pkg/pubsub/adapters/memory/memory.go` — adapter implementation
- `pkg/pubsub/adapters/memory/memory_test.go` — conformance + memory-specific tests

**Memory-specific tests**:
- `TestFanOut` — two subscribers each receive every message
- `TestIsolatedTopics` — messages on topic A not received on topic B
- `TestNoExternalDependencies` — zero config, no env vars

**Backward compat**: Purely additive. Existing `broker.go` in-memory path unchanged.

**Test order**: Write tests first, then implement.

---

### PR 4: NATS / JetStream Adapter

**What**: Unify the current `broker` (gocloud.dev natspubsub) and `NatsConnector` into a
single adapter that handles both core NATS and JetStream.

**Deliverables**:
- `pkg/pubsub/adapters/nats/nats.go` — adapter implementation
- `pkg/pubsub/adapters/nats/nats_test.go` — conformance + NATS-specific tests

**Config**:
```go
type Config struct {
    URLs       string `json:"urls"`
    JetStream  bool   `json:"jetstream"`
    StreamName string `json:"stream_name,omitempty"`
}
```

**NATS-specific tests** (build tag: `integration,nats`; CI uses `nats:latest -js`):
- `TestCoreNATSPublishSubscribe` — fire-and-forget
- `TestJetStreamDurableConsumer` — survives subscriber restart
- `TestQueueGroup` / `TestQueueGroupDifferentGroups` — competing vs fan-out
- `TestNATSHeaders` — header round-trip
- `TestJetStreamAckNack` — nack triggers redelivery
- `TestConnectionReconnect` — adapter recovers after disconnect
- `TestDrainOnClose` — buffered messages flushed

**Backward compat**: Purely additive. Existing `broker.go` NATS path and `broker/nats.go`
unchanged.

**Test order**: Write tests first, then implement.

---

### PR 5: AWS SQS Adapter

**What**: Replace the gocloud.dev `awssnssqs` backend with a direct `aws-sdk-go-v2`
implementation.

**Deliverables**:
- `pkg/pubsub/adapters/sqs/sqs.go` — adapter implementation
- `pkg/pubsub/adapters/sqs/sqs_test.go` — conformance + SQS-specific tests

**Config**:
```go
type Config struct {
    Region   string `json:"region"`
    QueueURL string `json:"queue_url"`
    TopicARN string `json:"topic_arn,omitempty"` // optional SNS fan-out
}
```

**SQS-specific tests** (build tag: `integration,sqs`; CI uses LocalStack):
- `TestSQSSendReceive` — basic round-trip
- `TestSQSDelaySeconds` — `Delay` → `DelaySeconds`
- `TestSQSFIFOMessageGroup` — `PartitionKey` → `MessageGroupId`
- `TestSQSVisibilityTimeout` — unacked message reappears
- `TestSQSRequeue` — `ChangeMessageVisibility`
- `TestSQSDeleteOnAck` — ack deletes message
- `TestSQSBatchReceive` — efficient batch polling
- `TestSQSLongPolling` — `WaitTimeSeconds`, no busy-loop
- `TestSNSPublish` — optional SNS topic publishing

**Backward compat**: Purely additive.

**Test order**: Write tests first, then implement.

---

### PR 6: GCP Pub/Sub Adapter

**What**: Replace the gocloud.dev `gcppubsub` backend with a direct
`cloud.google.com/go/pubsub` implementation.

**Deliverables**:
- `pkg/pubsub/adapters/gcppubsub/gcppubsub.go` — adapter implementation
- `pkg/pubsub/adapters/gcppubsub/gcppubsub_test.go` — conformance + GCP-specific tests

**Config**:
```go
type Config struct {
    ProjectID      string `json:"project_id"`
    SubscriptionID string `json:"subscription_id,omitempty"`
}
```

**GCP-specific tests** (build tag: `integration,gcppubsub`; CI uses GCP emulator):
- `TestGCPPublishSubscribe` — basic round-trip
- `TestGCPOrderingKey` — ordering key preserves order
- `TestGCPAckDeadline` — unacked redelivered after deadline
- `TestGCPMultipleSubscriptions` — different subscriptions each get all messages
- `TestGCPNack` — nack triggers redelivery
- `TestGCPAttributeRoundTrip` — Metadata ↔ attributes

**Backward compat**: Purely additive.

**Test order**: Write tests first, then implement.

---

### PR 7: RabbitMQ Adapter

**What**: New capability — RabbitMQ was not previously supported.

**Deliverables**:
- `pkg/pubsub/adapters/rabbitmq/rabbitmq.go` — adapter implementation
- `pkg/pubsub/adapters/rabbitmq/rabbitmq_test.go` — conformance + RabbitMQ-specific tests

**Config**:
```go
type Config struct {
    URL          string `json:"url"` // amqp://user:pass@host:5672/vhost
    ExchangeName string `json:"exchange_name"`
    ExchangeType string `json:"exchange_type"` // direct, topic, fanout, headers
    QueueName    string `json:"queue_name,omitempty"`
    Durable      bool   `json:"durable"`
}
```

**RabbitMQ-specific tests** (build tag: `integration,rabbitmq`; CI uses RabbitMQ container):
- `TestDirectExchange` — routing key exact match
- `TestTopicExchange` — wildcard routing
- `TestFanoutExchange` — all queues receive
- `TestHeaderExchange` — header-based routing
- `TestAckNackRequeue` — nack with/without requeue
- `TestDurableQueue` — survives subscriber restart
- `TestPrefetchCount` — `WithConcurrency` → QoS prefetch
- `TestConnectionRecovery` — auto-reconnect

**Backward compat**: Purely additive.

**Test order**: Write tests first, then implement.

---

### PR 8: Middleware (Logging, Metrics, Tracing)

**What**: Cross-cutting decorator middleware.

**Deliverables**:
- `pkg/pubsub/middleware/middleware.go` — `Middleware` type, `Compose` func
- `pkg/pubsub/middleware/logging.go` + `logging_test.go`
- `pkg/pubsub/middleware/metrics.go` + `metrics_test.go`
- `pkg/pubsub/middleware/tracing.go` + `tracing_test.go`

**Tests**:
- Logging: log on publish, subscribe start, errors
- Metrics: publish counter, error counter, latency histogram, topic labels
- Tracing: publish span, subscribe span, trace context propagation via headers

**Backward compat**: Purely additive.

**Test order**: Write tests first, then implement.

---

### PR 9: Wire In — Switch Constructors to Use Adapters + Bridge

**What**: Update `NewPublisher`/`NewSubscriber`/`NewPublishSubscriber` to internally create
new adapters + `LegacyBridge` instead of the old `broker`. This is the switchover PR.

**Deliverables**:
- Update `NewPublishSubscriber` to detect backend from `config.MessagingService` and create
  the corresponding adapter, wrapped in `LegacyBridge`
- Keep `broker.go` alive but unused (safety net for rollback)
- Register all adapters via blank imports

**Key constraint**: The function signatures of `NewPublisher`, `NewSubscriber`,
`NewPublishSubscriber` do NOT change. All existing callers compile and behave identically.

**Tests**:
- Existing `broker_test.go` tests must pass (they call `NewPublishSubscriber` → bridge →
  memory adapter)
- All integration tests for existing callers pass

**Backward compat**: Full. Same signatures, same behavior. Internal implementation changes only.

---

### PR 10: Cleanup — Remove gocloud.dev Dependency

**What**: Remove old code now that adapters are in production.

**Deliverables**:
- Delete `pkg/pubsub/broker.go` (old gocloud.dev broker)
- Delete `pkg/pubsub/broker/nats.go` (old NatsConnector — replaced by NATS adapter)
- Remove gocloud.dev imports and dependencies from `go.mod`
- Remove `TopicURLCreator` from `config/messaging.go` (or deprecate)
- Update `config.MessagingService` to carry `json.RawMessage` for adapter config

**Backward compat**: Breaking for `TopicURLCreator` users (internal only). All public
`Publisher`/`Subscriber`/`PublishSubscriber` interfaces remain stable.

---

## File Layout (final state after all PRs)

```
pkg/pubsub/
    pubsub.go               # Original interfaces: Publisher, Subscriber, PublishSubscriber,
                             #   PerformFunc, Message (preserved)
    adapter.go              # New Adapter interface
    handler.go              # HandlerFunc, Acknowledger
    options.go              # SubscribeConfig, SubscribeOption, option funcs
    registry.go             # Register(), NewAdapter()
    registry_test.go
    concurrency.go          # Shared semaphore-based concurrency helper
    concurrency_test.go
    bridge.go               # LegacyBridge (old interfaces → Adapter)
    bridge_test.go
    adaptertest/
        conformance.go      # Shared conformance test suite
    middleware/
        middleware.go        # Middleware type, Compose
        logging.go
        logging_test.go
        metrics.go
        metrics_test.go
        tracing.go
        tracing_test.go
    adapters/
        memory/
            memory.go
            memory_test.go
        kafka/
            kafka.go        # franz-go
            kafka_test.go
        nats/
            nats.go
            nats_test.go
        sqs/
            sqs.go
            sqs_test.go
        gcppubsub/
            gcppubsub.go
            gcppubsub_test.go
        rabbitmq/
            rabbitmq.go
            rabbitmq_test.go
```

## Open Questions

1. **Batch publish** — Should `Adapter` have a `PublishBatch(ctx, topic, []Message)` method?
   Kafka and SQS benefit significantly from batching. Could start without it and add later.
2. **Dead-letter topics** — Should DLQ config be part of `SubscribeConfig` or left to
   backend-specific configuration?
3. **Schema registry** — Kafka ecosystems often use schema registries (Avro/Protobuf). Should
   this be a concern of the adapter or a separate serialization layer?
4. **Message encoding** — The current TODO says "Let's NOT use JSON, please." Should we
   standardize on protobuf, or let callers choose the encoding and keep `Data` as `[]byte`?
