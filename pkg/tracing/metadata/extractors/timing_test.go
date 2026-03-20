package extractors

import (
	"testing"
	"time"

	"github.com/inngest/go-httpstat"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimingMetadata_Kind(t *testing.T) {
	md := TimingMetadata{}
	assert.Equal(t, metadata.Kind("inngest.timing"), md.Kind())
}

func TestTimingMetadata_Op(t *testing.T) {
	md := TimingMetadata{}
	assert.Equal(t, enums.MetadataOpcodeMerge, md.Op())
}

func TestTimingMetadata_Serialize(t *testing.T) {
	queueDelay := int64(150)
	systemLatency := int64(25)
	networkTotal := int64(200)
	totalInngest := int64(175)

	md := TimingMetadata{
		QueueDelayMs:    &queueDelay,
		SystemLatencyMs: &systemLatency,
		NetworkTotalMs:  &networkTotal,
		TotalInngestMs:  &totalInngest,
	}

	values, err := md.Serialize()
	require.NoError(t, err)
	assert.NotNil(t, values)
	assert.Contains(t, values, "queue_delay_ms")
	assert.Contains(t, values, "system_latency_ms")
	assert.Contains(t, values, "network_total_ms")
	assert.Contains(t, values, "total_inngest_ms")
}

func TestTimingMetadata_Serialize_OmitsNil(t *testing.T) {
	queueDelay := int64(100)
	md := TimingMetadata{
		QueueDelayMs: &queueDelay,
	}

	values, err := md.Serialize()
	require.NoError(t, err)
	assert.Contains(t, values, "queue_delay_ms")
	assert.NotContains(t, values, "network_total_ms")
}

func TestBuildTimingMetadata_WithRunInfoOnly(t *testing.T) {
	runInfo := &queue.RunInfo{
		SojournDelay: 150 * time.Millisecond,
		Latency:      25 * time.Millisecond,
	}

	md := BuildTimingMetadata(runInfo, nil)

	require.NotNil(t, md)
	require.NotNil(t, md.QueueDelayMs)
	assert.Equal(t, int64(150), *md.QueueDelayMs)

	require.NotNil(t, md.SystemLatencyMs)
	assert.Equal(t, int64(25), *md.SystemLatencyMs)

	require.NotNil(t, md.TotalInngestMs)
	assert.Equal(t, int64(175), *md.TotalInngestMs)

	assert.Nil(t, md.NetworkTotalMs)
}

func TestBuildTimingMetadata_WithHTTPStat(t *testing.T) {
	runInfo := &queue.RunInfo{
		SojournDelay: 100 * time.Millisecond,
		Latency:      20 * time.Millisecond,
	}

	stat := &httpstat.Result{
		Total: 200 * time.Millisecond,
	}

	md := BuildTimingMetadata(runInfo, stat)

	require.NotNil(t, md)
	require.NotNil(t, md.QueueDelayMs)
	assert.Equal(t, int64(100), *md.QueueDelayMs)

	require.NotNil(t, md.SystemLatencyMs)
	assert.Equal(t, int64(20), *md.SystemLatencyMs)

	require.NotNil(t, md.TotalInngestMs)
	assert.Equal(t, int64(120), *md.TotalInngestMs)

	require.NotNil(t, md.NetworkTotalMs)
	assert.Equal(t, int64(200), *md.NetworkTotalMs)
}

func TestBuildTimingMetadata_ZeroValues(t *testing.T) {
	runInfo := &queue.RunInfo{
		SojournDelay: 0,
		Latency:      0,
	}

	md := BuildTimingMetadata(runInfo, nil)

	require.NotNil(t, md)
	require.NotNil(t, md.QueueDelayMs)
	assert.Equal(t, int64(0), *md.QueueDelayMs)

	require.NotNil(t, md.SystemLatencyMs)
	assert.Equal(t, int64(0), *md.SystemLatencyMs)

	require.NotNil(t, md.TotalInngestMs)
	assert.Equal(t, int64(0), *md.TotalInngestMs)
}
