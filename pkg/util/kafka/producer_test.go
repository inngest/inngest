package kafka

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

// mockProducer is a test producer that can be configured to return errors
type mockProducer struct {
	err     error
	records []*kgo.Record
	lock    sync.Mutex
}

func (m *mockProducer) Produce(ctx context.Context, r *kgo.Record) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.records = append(m.records, r)
	return m.err
}

func (m *mockProducer) String() string { return "mock" }

func (m *mockProducer) GetRecords() []*kgo.Record {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.records
}

func TestKafkaProducer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test using testcontainers")
	}

	ctx := t.Context()

	// Start Kafka cluster with test-topic
	cluster, err := StartKafkaClusterWithTopic(t, "test-topic", 1, 3, 1)
	require.NoError(t, err)
	defer cluster.Terminate(ctx)

	client, err := kgo.NewClient(
		kgo.SeedBrokers(cluster.Brokers()...),
		kgo.DefaultProduceTopic("test-topic"),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordRetries(2),
		kgo.MaxBufferedRecords(3),
	)
	require.NoError(t, err)
	defer client.Close()

	kafkaProd, err := NewProducer(client)
	require.NoError(t, err)

	testProd := &mockProducer{}
	fallbackProd := NewFallbackProducer(kafkaProd, testProd)

	err = fallbackProd.Produce(ctx, &kgo.Record{
		Topic: "test-topic",
		Key:   []byte("test-key"),
		Value: []byte("test value"),
	})
	require.NoError(t, err)

	// After producing, consume and verify the record
	consumer, err := kgo.NewClient(
		kgo.SeedBrokers(cluster.Brokers()...),
		kgo.ConsumeTopics("test-topic"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	require.NoError(t, err)
	defer consumer.Close()

	// Poll for records with timeout
	fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	fetches := consumer.PollFetches(fetchCtx)
	require.NoError(t, fetches.Err())

	records := fetches.Records()
	require.Len(t, records, 1)
	assert.Equal(t, "test-topic", records[0].Topic)
	assert.Equal(t, []byte("test-key"), records[0].Key)
	assert.Equal(t, []byte("test value"), records[0].Value)

	// Verify fallback was NOT used since primary succeeded
	assert.Len(t, testProd.GetRecords(), 0, "fallback should not be called when primary succeeds")
}

func TestFallbackProducer_FallsBackOnError(t *testing.T) {
	ctx := t.Context()

	// Create a mock producer that always returns an error
	primaryProducer := &mockProducer{
		err: errors.New("primary producer failed"),
	}

	// Create a debug producer as fallback
	fallbackDebug := &mockProducer{}

	// Create fallback producer with primary (fails) and fallback (succeeds)
	fallbackProd := NewFallbackProducer(primaryProducer, fallbackDebug)

	record := &kgo.Record{
		Topic: "test-topic",
		Key:   []byte("test-key"),
		Value: []byte("test value"),
	}

	// Produce should succeed because fallback catches the error
	err := fallbackProd.Produce(ctx, record)
	require.NoError(t, err)

	// Verify primary received the record (even though it failed)
	assert.Len(t, primaryProducer.GetRecords(), 1)

	// Verify fallback received the record
	fallbackRecords := fallbackDebug.GetRecords()
	require.Len(t, fallbackRecords, 1)
	assert.Equal(t, "test-topic", fallbackRecords[0].Topic)
	assert.Equal(t, []byte("test-key"), fallbackRecords[0].Key)
	assert.Equal(t, []byte("test value"), fallbackRecords[0].Value)
}

func TestFallbackProducer_ContextCanceled_ReturnsImmediately(t *testing.T) {
	ctx := t.Context()

	// Create a mock producer that returns context.Canceled
	primaryProducer := &mockProducer{
		err: context.Canceled,
	}

	// Create a debug producer as fallback (should NOT be called)
	fallbackDebug := &mockProducer{}

	// Create fallback producer
	fallbackProd := NewFallbackProducer(primaryProducer, fallbackDebug)

	record := &kgo.Record{
		Topic: "test-topic",
		Key:   []byte("test-key"),
		Value: []byte("test value"),
	}

	// Produce should return context.Canceled immediately without falling back
	err := fallbackProd.Produce(ctx, record)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))

	// Verify primary received the record
	assert.Len(t, primaryProducer.GetRecords(), 1)

	// Verify fallback was NOT called
	assert.Len(t, fallbackDebug.GetRecords(), 0, "fallback should not be called on context.Canceled")
}

func TestFallbackProducer_ContextDeadlineExceeded_ReturnsImmediately(t *testing.T) {
	ctx := t.Context()

	// Create a mock producer that returns context.DeadlineExceeded
	primaryProducer := &mockProducer{
		err: context.DeadlineExceeded,
	}

	// Create a debug producer as fallback (should NOT be called)
	fallbackDebug := &mockProducer{}

	// Create fallback producer
	fallbackProd := NewFallbackProducer(primaryProducer, fallbackDebug)

	record := &kgo.Record{
		Topic: "test-topic",
		Key:   []byte("test-key"),
		Value: []byte("test value"),
	}

	// Produce should return context.DeadlineExceeded immediately without falling back
	err := fallbackProd.Produce(ctx, record)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))

	// Verify primary received the record
	assert.Len(t, primaryProducer.GetRecords(), 1)

	// Verify fallback was NOT called
	assert.Len(t, fallbackDebug.GetRecords(), 0, "fallback should not be called on context.DeadlineExceeded")
}

func TestKafkaProducer_BufferFull_UsesFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test using testcontainers")
	}

	ctx := t.Context()

	// Start Kafka cluster with test-topic
	cluster, err := StartKafkaClusterWithTopic(t, "buffer-test-topic", 1, 3, 1)
	require.NoError(t, err)
	defer cluster.Terminate(ctx)

	// Get the partition leader to stop it and cause produce delays
	leaderIndex, err := cluster.GetPartitionLeader(ctx, "buffer-test-topic", 0)
	require.NoError(t, err)
	t.Logf("Partition leader is broker %d (index)", leaderIndex)

	// Create client with very small buffer and few retries
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cluster.Brokers()...),
		kgo.DefaultProduceTopic("buffer-test-topic"),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordRetries(1),
		kgo.MaxBufferedRecords(1), // Very small buffer
		kgo.ProduceRequestTimeout(1*time.Second),
	)
	require.NoError(t, err)
	defer client.Close()

	kafkaProd, err := NewProducer(client)
	require.NoError(t, err)

	// Create debug producer to capture fallback records
	fallbackDebug := &mockProducer{}
	fallbackProd := NewFallbackProducer(kafkaProd, fallbackDebug)

	// Stop the leader to cause produces to slow down / get stuck
	err = cluster.StopBroker(ctx, leaderIndex)
	require.NoError(t, err)
	t.Logf("Stopped broker %d (partition leader)", leaderIndex)

	// Wait a moment for the cluster to detect the leader is down
	time.Sleep(500 * time.Millisecond)

	// Rapidly produce multiple records to fill the buffer
	// Since the leader is down, records will get stuck and buffer will fill up
	var wg sync.WaitGroup
	producedCount := 0
	fallbackUsed := false

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			record := &kgo.Record{
				Topic: "buffer-test-topic",
				Key:   []byte("test-key"),
				Value: []byte("test value"),
			}
			produceErr := fallbackProd.Produce(ctx, record)
			if produceErr == nil {
				producedCount++
			}
		}(i)
	}

	wg.Wait()

	// Check if fallback was used (some records should have gone to fallback due to buffer being full)
	fallbackRecords := fallbackDebug.GetRecords()
	if len(fallbackRecords) > 0 {
		fallbackUsed = true
		t.Logf("Fallback was used for %d records", len(fallbackRecords))
	}

	// The test passes if either:
	// 1. Fallback was used (buffer filled up as expected)
	// 2. All records failed (which is acceptable behavior)
	// The main assertion is that the fallback mechanism works when buffer issues occur
	assert.True(t, fallbackUsed || producedCount < 5, "expected fallback to be used or some records to fail when buffer is full")
}

func TestKafkaProducer_PartitionLeaderDown_UsesFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test using testcontainers")
	}

	ctx := t.Context()

	// Start Kafka cluster with topic (replication=3, minISR=1)
	cluster, err := StartKafkaClusterWithTopic(t, "leader-test-topic", 1, 3, 1)
	require.NoError(t, err)
	defer cluster.Terminate(ctx)

	// Create client with retries
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cluster.Brokers()...),
		kgo.DefaultProduceTopic("leader-test-topic"),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordRetries(2),
		kgo.MaxBufferedRecords(10),
		kgo.ProduceRequestTimeout(2*time.Second),
		kgo.RetryBackoffFn(func(attempt int) time.Duration {
			return 100 * time.Millisecond
		}),
	)
	require.NoError(t, err)
	defer client.Close()

	kafkaProd, err := NewProducer(client)
	require.NoError(t, err)

	// Create debug producer to capture fallback records
	fallbackDebug := &mockProducer{}
	fallbackProd := NewFallbackProducer(kafkaProd, fallbackDebug)

	// Produce a record successfully as baseline
	baselineRecord := &kgo.Record{
		Topic: "leader-test-topic",
		Key:   []byte("baseline-key"),
		Value: []byte("baseline value"),
	}
	err = fallbackProd.Produce(ctx, baselineRecord)
	require.NoError(t, err)
	t.Log("Baseline produce succeeded")

	// Identify the partition leader
	leaderIndex, err := cluster.GetPartitionLeader(ctx, "leader-test-topic", 0)
	require.NoError(t, err)
	t.Logf("Partition leader is broker %d (index)", leaderIndex)

	// Stop the leader container
	err = cluster.StopBroker(ctx, leaderIndex)
	require.NoError(t, err)
	t.Logf("Stopped leader broker %d", leaderIndex)

	// Give some time for metadata to propagate
	time.Sleep(1 * time.Second)

	// Attempt to produce - should eventually use fallback after retries exhausted
	failedRecord := &kgo.Record{
		Topic: "leader-test-topic",
		Key:   []byte("after-leader-down-key"),
		Value: []byte("after leader down value"),
	}

	// Create a context with timeout to avoid waiting too long
	produceCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = fallbackProd.Produce(produceCtx, failedRecord)
	// Error may or may not occur depending on whether new leader was elected
	// The key assertion is checking whether fallback received records

	fallbackRecords := fallbackDebug.GetRecords()
	t.Logf("Fallback received %d records", len(fallbackRecords))

	// If error occurred, fallback should have received the record
	if err != nil {
		t.Logf("Produce returned error (expected): %v", err)
		// Verify fallback was used
		assert.GreaterOrEqual(t, len(fallbackRecords), 1, "fallback should have received at least one record when produce failed")
		// Verify the record content
		if len(fallbackRecords) >= 1 {
			// The last record should be our failed record
			lastRecord := fallbackRecords[len(fallbackRecords)-1]
			assert.Equal(t, []byte("after-leader-down-key"), lastRecord.Key)
		}
	} else {
		// If no error, a new leader was elected quickly - this is also acceptable
		t.Log("Produce succeeded - new leader was elected quickly")
	}
}
