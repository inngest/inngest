package metrics

import (
	"github.com/eko/gocache/lib/v4/codec"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	defaultNamespace = "cache"
)

// Prometheus represents the prometheus struct for collecting metrics
type Prometheus struct {
	service      string
	namespace    string
	collector    *prometheus.GaugeVec
	registerer   prometheus.Registerer
	codecChannel chan codec.CodecInterface
}

// PrometheusOption is a type for defining Prometheus options
type PrometheusOption func(*Prometheus)

// WithCodecChannel sets the prometheus codec channel
func WithCodecChannel(codecChannel chan codec.CodecInterface) PrometheusOption {
	return func(m *Prometheus) {
		m.codecChannel = codecChannel
	}
}

// WithNamespace sets the prometheus namespace
func WithNamespace(namespace string) PrometheusOption {
	return func(m *Prometheus) {
		m.namespace = namespace
	}
}

// WithRegisterer sets the prometheus registerer
func WithRegisterer(registerer prometheus.Registerer) PrometheusOption {
	return func(m *Prometheus) {
		m.registerer = registerer
	}
}

// NewPrometheus initializes a new prometheus metric instance
func NewPrometheus(service string, options ...PrometheusOption) *Prometheus {
	instance := &Prometheus{
		namespace:    defaultNamespace,
		registerer:   prometheus.DefaultRegisterer,
		service:      service,
		codecChannel: make(chan codec.CodecInterface, 10000),
	}

	for _, option := range options {
		option(instance)
	}

	instance.collector = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "collector",
			Namespace: instance.namespace,
			Help:      "This represent the number of items in cache",
		},
		[]string{"service", "store", "metric"},
	)

	instance.registerer.MustRegister(instance.collector)

	go instance.recorder()

	return instance
}

// Record records a metric in prometheus by specifying the store name, metric name and value
func (m *Prometheus) record(store, metric string, value float64) {
	m.collector.WithLabelValues(m.service, store, metric).Set(value)
}

// Recorder records metrics in prometheus by retrieving values from the codec channel
func (m *Prometheus) recorder() {
	for codec := range m.codecChannel {
		stats := codec.GetStats()
		storeType := codec.GetStore().GetType()

		m.record(storeType, "hit_count", float64(stats.Hits))
		m.record(storeType, "miss_count", float64(stats.Miss))

		m.record(storeType, "set_success", float64(stats.SetSuccess))
		m.record(storeType, "set_error", float64(stats.SetError))

		m.record(storeType, "delete_success", float64(stats.DeleteSuccess))
		m.record(storeType, "delete_error", float64(stats.DeleteError))

		m.record(storeType, "invalidate_success", float64(stats.InvalidateSuccess))
		m.record(storeType, "invalidate_error", float64(stats.InvalidateError))
	}
}

// RecordFromCodec sends the given codec into the codec channel to be read from recorder
func (m *Prometheus) RecordFromCodec(codec codec.CodecInterface) {
	m.codecChannel <- codec
}
