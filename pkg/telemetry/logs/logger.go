package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/inngest/inngest/pkg/logger"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
	"log/slog"
	"time"
)

/*
	This package connects our logging libraries with a Kafka exporter, using the OTLP Protobuf definitions for logs.

	Logger (slog.Logger) -- Info() --> Handler

	Handler --> SplitHandler --> text handler (stdout/stderr)
							 --> exporter handler

	Exporter handler -- Translate logs --> OTel logger

	OTel logger -- Emit logs --> OTel simple processor -- Export --> Exporter

	Exporter -- Produce --> Kafka
*/

// NewKafkaLogger returns a split logger.Logger given a Kafka exporter.
// The split logger will emit logs to both the existing logger and exporter.
func NewKafkaLogger(
	ctx context.Context,
	existing logger.Logger,
	exporter log.Exporter,
) logger.Logger {
	// Translate slog records to OTel, emit, then send to exporter
	exporterHandler := newExporterHandler(exporter)

	// Push logs to existing handler
	split := logger.NewSplitHandler(
		existing.Handler(), // Log to stdout/stderr
		exporterHandler,    // Push logs to Kafka
	)

	// Create new inngest logger with export handler
	return logger.FromSlog(slog.New(split), existing.Level())
}

// exporterHandler is a translation layer between slog.Logger and OTel.
// It receives an OTel logger instance and will emit logs.
type exporterHandler struct {
	logger otellog.Logger
	level  slog.Level

	attrs []slog.Attr
	group string
}

func newExporterHandler(exporter log.Exporter) slog.Handler {
	// Immediately export records (add to Kafka produce batch)
	processor := log.NewSimpleProcessor(exporter)

	// Create scoped logger on resource
	otelLogger := log.
		NewLoggerProvider(log.WithProcessor(processor)).
		Logger("logger")

	return &exporterHandler{
		logger: otelLogger,
	}
}

func (e exporterHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= e.level
}

func (e exporterHandler) Handle(ctx context.Context, record slog.Record) error {
	exportRecord := otellog.Record{}
	exportRecord.SetTimestamp(record.Time)
	exportRecord.SetObservedTimestamp(time.Now())

	sev := convertSeverity(record.Level)
	exportRecord.SetSeverity(sev)
	exportRecord.SetSeverityText(sev.String())

	attrs := make([]otellog.KeyValue, 0)
	eventName := ""
	record.Attrs(func(attr slog.Attr) bool {
		// extract event name
		if attr.Key == logger.LoggerEventName && attr.Value.Kind() == slog.KindString {
			eventName = attr.Value.String()
			return true
		}

		attrs = append(attrs, otellog.KeyValue{
			Key:   attr.Key,
			Value: convertAttr(attr.Value),
		})
		return true
	})

	exportRecord.AddAttributes(attrs...)
	exportRecord.SetBody(otellog.StringValue(record.Message))

	if eventName != "" {
		exportRecord.SetEventName(eventName)
	}

	e.logger.Emit(ctx, exportRecord)

	return nil
}

func convertAttr(attr slog.Value) otellog.Value {
	switch attr.Kind() {
	case slog.KindString:
		return otellog.StringValue(attr.String())
	case slog.KindBool:
		return otellog.BoolValue(attr.Bool())
	case slog.KindFloat64:
		return otellog.Float64Value(attr.Float64())
	case slog.KindInt64:
		return otellog.Int64Value(attr.Int64())
	case slog.KindUint64:
		return otellog.Int64Value(int64(attr.Uint64()))
	case slog.KindDuration:
		return otellog.StringValue(attr.Duration().String())
	case slog.KindTime:
		return otellog.StringValue(attr.Time().Format(time.RFC3339))
	case slog.KindGroup:
		group := attr.Group()
		attrs := make([]otellog.KeyValue, len(group))
		for i, a := range group {
			attrs[i] = otellog.KeyValue{
				Key:   a.Key,
				Value: convertAttr(a.Value),
			}
		}
		return otellog.MapValue(attrs...)
	case slog.KindLogValuer:
		lv := attr.LogValuer()
		val := lv.LogValue()

		return convertAttr(val)
	case slog.KindAny:
		marshaled, err := json.Marshal(attr.Any())
		if err != nil {
			return otellog.StringValue("invalid_json")
		}

		return otellog.StringValue(string(marshaled))
	default:
		return otellog.StringValue(fmt.Sprintf("invalid kind %q", attr.Kind().String()))
	}

}

func convertSeverity(level slog.Level) otellog.Severity {
	switch level {
	case logger.LevelTrace:
		return otellog.SeverityTrace
	case logger.LevelDebug:
		return otellog.SeverityDebug
	case logger.LevelInfo:
		return otellog.SeverityInfo
	case logger.LevelNotice:
		return otellog.SeverityInfo1 // slightly above info but below warn
	case logger.LevelWarning:
		return otellog.SeverityWarn
	case logger.LevelError:
		return otellog.SeverityError
	case logger.LevelEmergency:
		return otellog.SeverityFatal
	}

	return otellog.SeverityUndefined
}

func (e exporterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	mergedAttributes := make([]slog.Attr, 0, len(e.attrs)+len(attrs))
	known := make(map[string]struct{})
	for _, attr := range attrs {
		mergedAttributes = append(mergedAttributes, attr)
		known[attr.Key] = struct{}{}
	}

	// filter out overwritten values in old attrs
	for _, attr := range e.attrs {
		if _, ok := known[attr.Key]; ok {
			continue
		}
		mergedAttributes = append(mergedAttributes, attr)
	}

	return exporterHandler{
		logger: e.logger,
		level:  e.level,
		attrs:  mergedAttributes,
	}
}

func (e exporterHandler) WithGroup(name string) slog.Handler {
	return exporterHandler{
		logger: e.logger,
		level:  e.level,
		attrs:  e.attrs,
		group:  name,
	}
}
