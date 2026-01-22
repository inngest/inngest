package kafka

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestRecordToKgoRecord(t *testing.T) {
	ts := time.Now().Truncate(time.Millisecond)
	record := &Record{
		Topic: "test-topic",
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
		Headers: []RecordHeader{
			{Key: "header-key", Value: []byte("header-value")},
		},
		Timestamp: ts,
	}

	kgoRecord := record.ToKgoRecord()

	assert.Equal(t, record.Topic, kgoRecord.Topic)
	assert.Equal(t, record.Key, kgoRecord.Key)
	assert.Equal(t, record.Value, kgoRecord.Value)
	assert.Len(t, kgoRecord.Headers, 1)
	assert.Equal(t, "header-key", kgoRecord.Headers[0].Key)
	assert.Equal(t, []byte("header-value"), kgoRecord.Headers[0].Value)
	assert.Equal(t, ts, kgoRecord.Timestamp)
}

func TestRecordToKgoRecord_NilFields(t *testing.T) {
	record := &Record{
		Topic: "test-topic",
		Key:   nil,
		Value: nil,
	}

	kgoRecord := record.ToKgoRecord()

	assert.Equal(t, "test-topic", kgoRecord.Topic)
	assert.Nil(t, kgoRecord.Key)
	assert.Nil(t, kgoRecord.Value)
	assert.Nil(t, kgoRecord.Headers)
	assert.True(t, kgoRecord.Timestamp.IsZero())
}

func TestRecordFromKgo(t *testing.T) {
	ts := time.Now().Truncate(time.Millisecond)
	kgoRecord := &kgo.Record{
		Topic: "test-topic",
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
		Headers: []kgo.RecordHeader{
			{Key: "header-key", Value: []byte("header-value")},
		},
		Timestamp: ts,
	}

	record := RecordFromKgo(kgoRecord)

	assert.Equal(t, kgoRecord.Topic, record.Topic)
	assert.Equal(t, kgoRecord.Key, record.Key)
	assert.Equal(t, kgoRecord.Value, record.Value)
	assert.Len(t, record.Headers, 1)
	assert.Equal(t, "header-key", record.Headers[0].Key)
	assert.Equal(t, []byte("header-value"), record.Headers[0].Value)
	assert.Equal(t, ts, record.Timestamp)
}

func TestRecordFromKgo_NilFields(t *testing.T) {
	kgoRecord := &kgo.Record{
		Topic: "test-topic",
		Key:   nil,
		Value: nil,
	}

	record := RecordFromKgo(kgoRecord)

	assert.Equal(t, "test-topic", record.Topic)
	assert.Nil(t, record.Key)
	assert.Nil(t, record.Value)
	assert.Nil(t, record.Headers)
	assert.True(t, record.Timestamp.IsZero())
}

func TestRecordRoundTrip(t *testing.T) {
	ts := time.Now().Truncate(time.Millisecond)
	original := &Record{
		Topic: "round-trip-topic",
		Key:   []byte("round-trip-key"),
		Value: []byte("round-trip-value"),
		Headers: []RecordHeader{
			{Key: "h1", Value: []byte("v1")},
			{Key: "h2", Value: []byte("v2")},
		},
		Timestamp: ts,
	}

	// Convert to kgo.Record and back
	kgoRecord := original.ToKgoRecord()
	roundTripped := RecordFromKgo(kgoRecord)

	assert.Equal(t, original.Topic, roundTripped.Topic)
	assert.Equal(t, original.Key, roundTripped.Key)
	assert.Equal(t, original.Value, roundTripped.Value)
	assert.Equal(t, original.Headers, roundTripped.Headers)
	assert.Equal(t, original.Timestamp, roundTripped.Timestamp)
}

func TestRecordRoundTrip_EmptySlices(t *testing.T) {
	original := &Record{
		Topic: "empty-topic",
		Key:   []byte{},
		Value: []byte{},
	}

	// Convert to kgo.Record and back
	kgoRecord := original.ToKgoRecord()
	roundTripped := RecordFromKgo(kgoRecord)

	assert.Equal(t, original.Topic, roundTripped.Topic)
	assert.Equal(t, original.Key, roundTripped.Key)
	assert.Equal(t, original.Value, roundTripped.Value)
}

func TestRecordToKgoRecord_MultipleHeaders(t *testing.T) {
	record := &Record{
		Topic: "test-topic",
		Headers: []RecordHeader{
			{Key: "content-type", Value: []byte("application/json")},
			{Key: "trace-id", Value: []byte("abc123")},
			{Key: "empty-value", Value: nil},
		},
	}

	kgoRecord := record.ToKgoRecord()

	assert.Len(t, kgoRecord.Headers, 3)
	assert.Equal(t, "content-type", kgoRecord.Headers[0].Key)
	assert.Equal(t, []byte("application/json"), kgoRecord.Headers[0].Value)
	assert.Equal(t, "trace-id", kgoRecord.Headers[1].Key)
	assert.Equal(t, []byte("abc123"), kgoRecord.Headers[1].Value)
	assert.Equal(t, "empty-value", kgoRecord.Headers[2].Key)
	assert.Nil(t, kgoRecord.Headers[2].Value)
}
