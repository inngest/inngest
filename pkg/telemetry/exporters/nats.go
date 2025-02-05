package exporters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/pubsub/broker"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	runv2 "github.com/inngest/inngest/proto/gen/run/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/protobuf/proto"
)

// NATS span exporter
type natsSpanExporter struct {
	streams    []*StreamConf
	conn       *broker.NatsConnector
	deadletter *StreamConf
	// buffer to be used to store spans temporarily if nats client is overwhelmed
	buf *natsBuffer
}

type natsSpanExporterOpts struct {
	streams []*StreamConf
	// Comma delimited URLs of the NATS server to use
	urls string
	// The path of the nkey file to be used for authentication
	nkeyFile string
	// The credentials file to be used for authentication
	credsFile string
	// The deadletter stream to send the failed attempts to
	deadletter *StreamConf
	// The number of goroutines to handle deadletter delivery
	// defaults to 100
	dlConcurrency int
	// The buffer to be used for the deadletter channel
	// defaults to 10,000
	dlcBuffer int
}

type StreamConf struct {
	// Subject of the NATS Stream
	Subject string
}

type NatsExporterOpts func(n *natsSpanExporterOpts)

func WithNatsExporterStream(stream StreamConf) NatsExporterOpts {
	return func(n *natsSpanExporterOpts) {
		n.streams = append(n.streams, &stream)
	}
}

func WithNatsExporterUrls(urls string) NatsExporterOpts {
	return func(n *natsSpanExporterOpts) {
		n.urls = urls
	}
}

func WithNatsExporterNKeyFile(nkeyFilePath string) NatsExporterOpts {
	return func(n *natsSpanExporterOpts) {
		n.nkeyFile = nkeyFilePath
	}
}

func WithNatsExporterCredsFile(credsFilePath string) NatsExporterOpts {
	return func(n *natsSpanExporterOpts) {
		n.credsFile = credsFilePath
	}
}

func WithNatsExporterDeadLetter(s StreamConf) NatsExporterOpts {
	return func(n *natsSpanExporterOpts) {
		n.deadletter = &s
	}
}

// NewNATSSpanExporter creates an otel compatible exporter that ships the spans to NATS
func NewNATSSpanExporter(ctx context.Context, opts ...NatsExporterOpts) (trace.SpanExporter, error) {
	if len(opts) == 0 {
		return nil, fmt.Errorf("no nats exporter options provided")
	}

	expOpts := &natsSpanExporterOpts{
		streams:       []*StreamConf{},
		dlConcurrency: 100,
		dlcBuffer:     10_000,
	}
	for _, apply := range opts {
		apply(expOpts)
	}

	connOpts := []nats.Option{}
	// attempt to parse nkey file is the option was passed in
	if expOpts.nkeyFile != "" {
		auth, err := nats.NkeyOptionFromSeed(expOpts.nkeyFile)
		if err != nil {
			return nil, fmt.Errorf("error parsing nkey file for NATS: %w", err)
		}
		connOpts = append(connOpts, auth)
	}

	// Use chain credentials file for auth
	if expOpts.credsFile != "" {
		auth := nats.UserCredentials(expOpts.credsFile)
		connOpts = append(connOpts, auth)
	}

	conn, err := broker.NewNATSConnector(ctx, broker.NatsConnOpt{
		Name:      "run-span-exporter",
		URLS:      expOpts.urls,
		JetStream: true,
		Opts:      connOpts,
	})
	if err != nil {
		return nil, fmt.Errorf("error setting up nats: %w", err)
	}

	exporter := &natsSpanExporter{
		conn:    conn,
		streams: expOpts.streams,
		buf:     newNatsBuffer(),
	}

	go exporter.handleBuffered(ctx)

	return exporter, nil
}

func (e *natsSpanExporter) handleBuffered(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			// create a new context so the current one doesn't get cancelled
			ctx = context.Background()

			if err := e.flush(ctx, e.streams, false); err != nil {
				logger.StdlibLogger(ctx).Error("error flushing to NATS streams",
					"error", err,
					"streams", e.streams,
				)
			}

			dls := []*StreamConf{e.deadletter}
			if err := e.flush(ctx, dls, true); err != nil {
				logger.StdlibLogger(ctx).Error("error flushing to NATS deadletter stream",
					"error", err,
					"streams", dls,
				)
			}

			ticker.Stop()
			return

		case <-ticker.C:
			if e.deadletter == nil {
				continue
			}

			dls := []*StreamConf{e.deadletter}
			if err := e.flush(ctx, dls, true); err != nil {
				logger.StdlibLogger(ctx).Error("error flushing to NATS deadletter stream",
					"error", err,
					"streams", dls,
				)
			}
		}
	}
}

func (e *natsSpanExporter) flush(ctx context.Context, streams []*StreamConf, deadletter bool) error {
	if len(streams) == 0 {
		// no op
		return nil
	}

	js, err := e.conn.JSConn()
	if err != nil {
		return err
	}
	wg := sync.WaitGroup{}

	retries := e.buf.Retrieve()
	size := len(retries)
	metrics.GaugeSpanExporterBuffer(ctx, int64(size), metrics.GaugeOpt{PkgName: pkgName})
	if size == 0 {
		// no op
		return nil
	}

	for _, stream := range streams {
		for _, sr := range retries {
			wg.Add(1)
			go func(ctx context.Context, conf StreamConf, sr *spanRetry) {
				defer wg.Done()

				if deadletter {
					metrics.IncrSpanBatchProcessorDeadLetterCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
				}

				id := sr.span.Id
				byt, err := proto.Marshal(sr.span)
				if err != nil {
					logger.StdlibLogger(ctx).Error("error serializing span to protobuf",
						"error", err,
						"stream", conf.Subject,
						"acctID", id.AccountId,
						"wsID", id.EnvId,
						"wfID", id.FunctionId,
						"runID", id.RunId,
						"retry", sr,
					)

					return
				}

				fack, err := js.PublishAsync(conf.Subject, byt,
					jetstream.WithStallWait(1*time.Second),
					jetstream.WithRetryAttempts(20),
				)
				if err != nil {
					logger.StdlibLogger(ctx).Error("error on async publish to nats stream",
						"error", err,
						"stream", conf.Subject,
						"acctID", id.AccountId,
						"wsID", id.EnvId,
						"wfID", id.FunctionId,
						"runID", id.RunId,
						"retry", sr,
					)

					// publish it back again for retries
					sr.attempt++
					e.buf.Add(sr)
					return
				}

				status := "unknown"
				select {
				case <-fack.Ok():
					status = "success"
				case err := <-fack.Err():
					status = "error"

					logger.StdlibLogger(ctx).Error("error with async publish to nats stream",
						"error", err,
						"stream", conf.Subject,
						"acctID", id.AccountId,
						"wsID", id.EnvId,
						"wfID", id.FunctionId,
						"runID", id.RunId,
						"retry", sr,
					)

					// publish it back again for retries
					sr.attempt++
					e.buf.Add(sr)
				}

				metrics.IncrSpanExportedCounter(ctx, metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"subject": conf.Subject,
						"status":  status,
					},
				})
				if deadletter {
					metrics.IncrSpanBatchProcessorDeadLetterPublishStatusCounter(ctx, metrics.CounterOpt{
						PkgName: pkgName,
						Tags: map[string]any{
							"status": status,
							"stream": conf.Subject,
						},
					})
				}
			}(ctx, *stream, sr)
		}
	}

	wg.Wait()
	return nil
}

func (e *natsSpanExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	ctx = context.WithoutCancel(ctx)
	wg := sync.WaitGroup{}

	// Expect jetstream to be enabled
	js, err := e.conn.JSConn()
	if err != nil {
		return err
	}
	// publish to all subjects defined
	for _, stream := range e.streams {
		for _, sp := range spans {
			wg.Add(1)

			go func(ctx context.Context, conf StreamConf, sp trace.ReadOnlySpan) {
				defer wg.Done()

				span, err := SpanToProto(ctx, sp)
				if err != nil {
					return
				}
				id := span.Id

				pending := js.PublishAsyncPending()
				metrics.GaugeSpanBatchProcessorNatsAsyncPending(ctx, int64(pending), metrics.GaugeOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"subject": conf.Subject,
					},
				})
				if pending >= e.conn.BufferSize {
					// don't try to send because it'll likely stall
					e.buf.Add(&spanRetry{span: span})
					return
				}

				// serialize it into bytes
				byt, err := proto.Marshal(span)
				if err != nil {
					logger.StdlibLogger(ctx).Error("error serializing span to protobuf",
						"error", err,
						"stream", conf.Subject,
						"acctID", id.AccountId,
						"wsID", id.EnvId,
						"wfID", id.FunctionId,
						"runID", id.RunId,
					)
					return
				}

				// Use async publish to increase throughput
				fack, err := js.PublishAsync(conf.Subject, byt,
					jetstream.WithStallWait(500*time.Millisecond),
					jetstream.WithRetryAttempts(10),
				)
				if err != nil {
					logger.StdlibLogger(ctx).Error("error on async publish to nats stream",
						"error", err,
						"stream", conf.Subject,
						"acctID", id.AccountId,
						"wsID", id.EnvId,
						"wfID", id.FunctionId,
						"runID", id.RunId,
					)

					e.buf.Add(&spanRetry{span: span})
					return
				}

				pstatus := "unknown"
				select {
				case <-fack.Ok():
					pstatus = "success"
				case err := <-fack.Err():
					pstatus = "error"

					logger.StdlibLogger(ctx).Error("error with async publish to nats stream",
						"error", err,
						"stream", conf.Subject,
						"acctID", id.AccountId,
						"wsID", id.EnvId,
						"wfID", id.FunctionId,
						"runID", id.RunId,
					)

					e.buf.Add(&spanRetry{span: span})
				}

				metrics.IncrSpanExportedCounter(ctx, metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"producer": "nats",
						"subject":  conf.Subject,
						"status":   pstatus,
					},
				})
			}(ctx, *stream, sp)
		}
	}

	wg.Wait()
	return nil
}

func (e *natsSpanExporter) Shutdown(ctx context.Context) error {
	logger.StdlibLogger(ctx).Info("shutting down nats span exporter")

	// create a new context so the current one doesn't get cancelled
	ctx = context.Background()

	if err := e.flush(ctx, e.streams, false); err != nil {
		logger.StdlibLogger(ctx).Error("error flushing to NATS streams",
			"error", err,
			"streams", e.streams,
		)
	}

	dls := []*StreamConf{e.deadletter}
	if err := e.flush(ctx, dls, true); err != nil {
		logger.StdlibLogger(ctx).Error("error flushing to NATS deadletter stream",
			"error", err,
			"streams", dls,
		)
	}

	return e.conn.Shutdown(ctx)
}

type spanRetry struct {
	span    *runv2.Span
	attempt int
}

func newNatsBuffer() *natsBuffer {
	return &natsBuffer{
		buf: []*spanRetry{},
	}
}

type natsBuffer struct {
	sync.Mutex

	buf []*spanRetry
}

func (b *natsBuffer) Add(s *spanRetry) {
	b.Lock()
	defer b.Unlock()

	b.buf = append(b.buf, s)
}

func (b *natsBuffer) Retrieve() []*spanRetry {
	b.Lock()
	defer b.Unlock()

	res := b.buf
	b.buf = []*spanRetry{}

	return res
}
