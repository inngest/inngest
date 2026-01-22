package kafka

import (
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// RecordHeader contains extra information that can be sent with Records.
type RecordHeader struct {
	Key   string
	Value []byte
}

// Record is a library-agnostic Kafka record for producing messages.
type Record struct {
	Topic     string
	Key       []byte
	Value     []byte
	Headers   []RecordHeader
	Timestamp time.Time
}

// ToKgoRecord converts a kafka.Record to a kgo.Record.
func (r *Record) ToKgoRecord() *kgo.Record {
	var headers []kgo.RecordHeader
	if len(r.Headers) > 0 {
		headers = make([]kgo.RecordHeader, len(r.Headers))
		for i, h := range r.Headers {
			headers[i] = kgo.RecordHeader{
				Key:   h.Key,
				Value: h.Value,
			}
		}
	}

	return &kgo.Record{
		Topic:     r.Topic,
		Key:       r.Key,
		Value:     r.Value,
		Headers:   headers,
		Timestamp: r.Timestamp,
	}
}

// RecordFromKgo converts a kgo.Record to a kafka.Record.
func RecordFromKgo(r *kgo.Record) *Record {
	var headers []RecordHeader
	if len(r.Headers) > 0 {
		headers = make([]RecordHeader, len(r.Headers))
		for i, h := range r.Headers {
			headers[i] = RecordHeader{
				Key:   h.Key,
				Value: h.Value,
			}
		}
	}

	return &Record{
		Topic:     r.Topic,
		Key:       r.Key,
		Value:     r.Value,
		Headers:   headers,
		Timestamp: r.Timestamp,
	}
}
