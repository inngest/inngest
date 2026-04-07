package extractors

import (
	"github.com/inngest/go-httpstat"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/tracing/metadata"
)

//tygo:generate
const (
	KindInngestTiming metadata.Kind = "inngest.timing"
)

// TimingMetadata contains high-level timing categories for a step execution:
// queue delay, system processing overhead, and network total.
//
//tygo:generate
type TimingMetadata struct {
	// QueueDelayMs is the sojourn delay caused by concurrency limits, throttle,
	// or other user-defined concurrency constraints.
	QueueDelayMs *int64 `json:"queue_delay_ms,omitempty"`
	// SystemLatencyMs is the processing delay excluding sojourn latency
	// (time from queue lease to execution start).
	SystemLatencyMs *int64 `json:"system_latency_ms,omitempty"`
	// NetworkTotalMs is the total HTTP request duration from httpstat,
	// covering the full SDK call lifecycle.
	NetworkTotalMs *int64 `json:"network_total_ms,omitempty"`
	// TotalInngestMs is the sum of Inngest-side overhead (queue delay + system latency).
	TotalInngestMs *int64 `json:"total_inngest_ms,omitempty"`
}

func (m TimingMetadata) Kind() metadata.Kind {
	return KindInngestTiming
}

func (m TimingMetadata) Op() metadata.Opcode {
	return enums.MetadataOpcodeMerge
}

func (m TimingMetadata) Serialize() (metadata.Values, error) {
	var rawMetadata metadata.Values
	err := rawMetadata.FromStruct(m)
	if err != nil {
		return nil, err
	}

	return rawMetadata, nil
}

// BuildTimingMetadata constructs a TimingMetadata from queue RunInfo and
// an optional httpstat result.
func BuildTimingMetadata(runInfo *queue.RunInfo, stat *httpstat.Result) *TimingMetadata {
	if runInfo == nil {
		return nil
	}

	md := &TimingMetadata{}

	queueDelayMs := runInfo.SojournDelay.Milliseconds()
	md.QueueDelayMs = &queueDelayMs

	systemLatencyMs := runInfo.Latency.Milliseconds()
	md.SystemLatencyMs = &systemLatencyMs

	totalInngestMs := queueDelayMs + systemLatencyMs
	md.TotalInngestMs = &totalInngestMs

	if stat != nil {
		networkTotalMs := stat.Total.Milliseconds()
		md.NetworkTotalMs = &networkTotalMs
	}

	return md
}
