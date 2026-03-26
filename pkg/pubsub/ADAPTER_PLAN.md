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
- Preserve backward compatibility during migration.

## Design

### Core Interfaces (`pkg/pubsub/adapter.go`)

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

### Message Type (`pkg/pubsub/message.go`)

```go
type Message struct {
    // Core fields (always used)
    ID        string            `json:"id,omitempty"`
    Name      string            `json:"name"`
    Version   int               `json:"v"`
    Data      []byte            `json:"data"`
    Timestamp time.Time         `json:"ts"`
    Metadata  map[string]string `json:"meta,omitempty"`

    // Backend hints — adapters use what they support, ignore the rest.
    // These are not serialized; they're set in-process before Publish().
    PartitionKey string        `json:"-"` // Kafka partition, SQS message group ID
    OrderingKey  string        `json:"-"` // GCP Pub/Sub ordering key
    RoutingKey   string        `json:"-"` // RabbitMQ routing key
    Delay        time.Duration `json:"-"` // SQS delay, RabbitMQ delayed message
    Headers      map[string]string `json:"-"` // Kafka headers, RabbitMQ headers, NATS headers
}
```

Key change: `Data` is `[]byte` instead of `string`. The encoding strategy (JSON, protobuf,
msgpack) is the caller's responsibility, not the adapter's.

### Handler and Ack (`pkg/pubsub/handler.go`)

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

The current design infers ack/nack from the handler's error return. Explicit ack gives handlers
control over partial processing, dead-letter routing, and delayed requeue.

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

### Adapter Registry (`pkg/pubsub/registry.go`)

```go
// AdapterFactory creates an adapter from backend-specific config.
type AdapterFactory func(ctx context.Context, cfg json.RawMessage) (Adapter, error)

var (
    mu       sync.RWMutex
    registry = map[string]AdapterFactory{}
)

// Register makes a backend available by name. Called from init().
func Register(name string, factory AdapterFactory) {
    mu.Lock()
    defer mu.Unlock()
    registry[name] = factory
}

// NewAdapter creates an adapter for the named backend.
func NewAdapter(ctx context.Context, backend string, cfg json.RawMessage) (Adapter, error) {
    mu.RLock()
    factory, ok := registry[backend]
    mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("unknown pubsub backend: %s", backend)
    }
    return factory(ctx, cfg)
}
```

Each adapter package registers itself:
```go
// In adapters/kafka/kafka.go
func init() {
    pubsub.Register("kafka", New)
}
```

### Middleware (`pkg/pubsub/middleware/`)

```go
type Middleware func(Adapter) Adapter

// Compose applies middleware in order (outermost first).
func Compose(adapter Adapter, mw ...Middleware) Adapter {
    for i := len(mw) - 1; i >= 0; i-- {
        adapter = mw[i](adapter)
    }
    return adapter
}
```

Initial middleware:
- **`WithLogging`** — log publish/subscribe events, errors.
- **`WithMetrics`** — publish/subscribe counters, latency histograms.
- **`WithTracing`** — OpenTelemetry span propagation.

### Concurrency Helper (`pkg/pubsub/concurrency.go`)

The weighted-semaphore pattern currently in `broker.go` is good. Extract it as a reusable
helper that adapters can optionally use:

```go
// ConcurrentSubscriber wraps a base receive loop with semaphore-based concurrency.
// Adapters that don't need custom concurrency can delegate to this.
func ConcurrentSubscriber(
    ctx context.Context,
    concurrency int,
    receive func(ctx context.Context) (Message, Acknowledger, error),
    handler HandlerFunc,
) error { ... }
```

## Adapter Implementations

### 1. In-Memory (`pkg/pubsub/adapters/memory/`)

- For testing and single-process dev server.
- Channel-based fan-out per topic.
- No external dependencies.

### 2. NATS / JetStream (`pkg/pubsub/adapters/nats/`)

- Unifies the current `broker` (via gocloud.dev) and `NatsConnector` into a single adapter.
- Core NATS for fire-and-forget, JetStream for durable subscriptions.
- Maps `ConsumerGroup` → NATS queue group / JetStream durable consumer.
- Maps `Headers` → NATS message headers.
- Uses existing `nats.go` and `nats.go/jetstream` packages.

Config:
```go
type Config struct {
    URLs       string `json:"urls"`
    JetStream  bool   `json:"jetstream"`
    StreamName string `json:"stream_name,omitempty"`
}
```

### 3. Kafka (`pkg/pubsub/adapters/kafka/`)

- Uses **franz-go** (`github.com/twmb/franz-go`) — most complete Kafka protocol implementation.
- Maps `PartitionKey` → Kafka record key.
- Maps `ConsumerGroup` → Kafka consumer group.
- Maps `Headers` → Kafka record headers.
- Maps `StartOffset` → `kgo.ConsumeResetOffset`.
- Supports batch consumption internally, dispatches to handler per-record.

Config:
```go
type Config struct {
    Brokers       []string `json:"brokers"`
    TLS           bool     `json:"tls"`
    SASLMechanism string   `json:"sasl_mechanism,omitempty"` // PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
    SASLUser      string   `json:"sasl_user,omitempty"`
    SASLPass      string   `json:"sasl_pass,omitempty"`
}
```

### 4. AWS SQS (`pkg/pubsub/adapters/sqs/`)

- Uses `aws-sdk-go-v2`.
- Maps `PartitionKey` → SQS `MessageGroupId` (FIFO queues).
- Maps `Delay` → `DelaySeconds`.
- Maps `Ack` → `DeleteMessage`, `Nack` → no-op (visibility timeout expires), `Requeue` → `ChangeMessageVisibility`.
- Publish goes to SNS topic or directly to SQS queue (configurable).

Config:
```go
type Config struct {
    Region   string `json:"region"`
    QueueURL string `json:"queue_url"`
    // Optional: publish to SNS topic instead of directly to SQS
    TopicARN string `json:"topic_arn,omitempty"`
}
```

### 5. GCP Pub/Sub (`pkg/pubsub/adapters/gcppubsub/`)

- Uses `cloud.google.com/go/pubsub`.
- Maps `OrderingKey` → GCP ordering key.
- Maps `ConsumerGroup` → GCP subscription name.
- Maps `Ack`/`Nack` → native GCP ack/nack.

Config:
```go
type Config struct {
    ProjectID      string `json:"project_id"`
    SubscriptionID string `json:"subscription_id,omitempty"`
}
```

### 6. RabbitMQ (`pkg/pubsub/adapters/rabbitmq/`)

- Uses `github.com/rabbitmq/amqp091-go`.
- Maps `RoutingKey` → AMQP routing key.
- Maps `Headers` → AMQP headers (for header-based routing).
- Maps `Ack`/`Nack`/`Requeue` → AMQP ack/nack/reject with requeue.
- Configurable exchange type (direct, topic, fanout, headers).

Config:
```go
type Config struct {
    URL          string `json:"url"` // amqp://user:pass@host:5672/vhost
    ExchangeName string `json:"exchange_name"`
    ExchangeType string `json:"exchange_type"` // direct, topic, fanout, headers
    QueueName    string `json:"queue_name,omitempty"`
    Durable      bool   `json:"durable"`
}
```

## Migration Strategy

### Phase 1: Introduce new interfaces alongside existing code
- Add `adapter.go`, `message.go`, `handler.go`, `options.go`, `registry.go`.
- Add `concurrency.go` helper extracted from current `broker.go`.
- No changes to existing `broker.go` or `config/messaging.go`.

### Phase 2: Implement adapters
- Start with `memory` and `nats` adapters (these cover existing test + production use).
- Add `kafka` adapter (new capability).
- Add `sqs` and `gcppubsub` adapters (replace gocloud.dev versions).
- Add `rabbitmq` adapter.

### Phase 3: Bridge layer for backward compatibility
- Create a `LegacyBridge` that wraps the new `Adapter` interface and exposes the old
  `Publisher`/`Subscriber`/`PublishSubscriber` interfaces.
- Update `NewPublisher()`, `NewSubscriber()`, `NewPublishSubscriber()` to use the bridge
  internally.
- Existing callers (`api/service.go`, `runner/runner.go`, `executor/service.go`,
  `devserver/devserver.go`) continue to work unchanged.

### Phase 4: Migrate callers
- Update callers to use `Adapter` directly.
- Remove bridge layer and old `broker.go`.
- Remove `gocloud.dev/pubsub` dependency.
- Remove `TopicURLCreator` interface from config.

## File Layout

```
pkg/pubsub/
    adapter.go              # Adapter interface
    message.go              # Message type with backend hints
    handler.go              # HandlerFunc, Acknowledger
    options.go              # SubscribeOption, SubscribeConfig
    registry.go             # Register(), NewAdapter()
    concurrency.go          # Shared semaphore-based concurrency helper
    bridge.go               # LegacyBridge for backward compat (Phase 3)
    middleware/
        logging.go
        metrics.go
        tracing.go
    adapters/
        memory/
            memory.go
            memory_test.go
        nats/
            nats.go         # Unifies current broker + NatsConnector
            nats_test.go
        kafka/
            kafka.go        # franz-go based
            kafka_test.go
        sqs/
            sqs.go
            sqs_test.go
        gcppubsub/
            gcppubsub.go
            gcppubsub_test.go
        rabbitmq/
            rabbitmq.go
            rabbitmq_test.go
    # Legacy (removed in Phase 4)
    broker.go               # Current gocloud.dev broker
    broker_test.go
    broker/
        nats.go             # Current NatsConnector
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
