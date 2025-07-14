package bench

import (
	"context"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/joho/godotenv"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"golang.org/x/sync/errgroup"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func freePort() int {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func setupMetrics(ctx context.Context, b *testing.B) func() {
	//col, err := otelcol.NewCollector(otelcol.CollectorSettings{
	//	Factories:               nil,
	//	BuildInfo:               component.BuildInfo{},
	//	DisableGracefulShutdown: false,
	//	ConfigProviderSettings: otelcol.ConfigProviderSettings{
	//		ResolverSettings: confmap.ResolverSettings{},
	//	},
	//	ProviderModules:       nil,
	//	ConverterModules:      nil,
	//	LoggingOptions:        nil,
	//	SkipSettingGRPCLogger: false,
	//})
	//require.NoError(b, err)
	//err = col.Run(ctx)
	//require.NoError(b, err)

	endpoint := os.Getenv("OTEL_METRICS_COLLECTOR_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4317"
	}

	// NOTE:
	// assuming the otel collector is within the same private network, we can
	// skip grpc auth, but probably still better to get it work for production
	//conn, err := grpc.NewClient(endpoint,
	//	grpc.WithTransportCredentials(insecure.NewCredentials()),
	//)
	//require.NoError(b,err)

	//exp, err := otlpmetricgrpc.New(ctx,
	//	otlpmetricgrpc.WithGRPCConn(conn),
	//)
	//require.NoError(b,err)
	//

	exp, err := otlpmetrichttp.New(ctx)
	require.NoError(b, err)

	reader := metric.NewPeriodicReader(exp)
	meter := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("benchmark"),
		)),
	)

	otel.SetMeterProvider(meter)

	//meterCloser, err := metrics.MeterSetup("benchmark", metrics.MeterTypeOTLP)
	//require.NoError(b, err)

	return func() {
		err := meter.Shutdown(context.Background())
		if err != nil {
			b.Logf("failed to shut down meter: %s\n", err.Error())
		}
	}
}

func setupTraces() {
	//port := freePort()
	//traceServer := http.Server{}
	//
	//traceServer
	//
	//traceEndpoint := fmt.Sprintf("localhost:%d", port)
	//if err := itrace.NewUserTracer(ctx, itrace.TracerOpts{
	//	ServiceName:   "tracing",
	//	TraceEndpoint: traceEndpoint,
	//	TraceURLPath:  "/dev/traces",
	//	Type:          itrace.TracerTypeOTLPHTTP,
	//}); err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
	//defer func() {
	//	_ = itrace.CloseUserTracer(ctx)
	//}()
	//
	//if err := itrace.NewSystemTracer(ctx, itrace.TracerOpts{
	//	ServiceName:   "tracing-system",
	//	TraceEndpoint: traceEndpoint,
	//	TraceURLPath:  "/dev/traces/system",
	//	Type:          itrace.TracerTypeOTLPHTTP,
	//}); err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
	//defer func() {
	//	_ = itrace.CloseSystemTracer(ctx)
	//}()
}

func BenchmarkQueue(b *testing.B) {
	ctx := context.Background()

	err := godotenv.Load()
	require.NoError(b, err)

	meterCloser := setupMetrics(ctx, b)
	defer meterCloser()

	processItems := 1000
	wg := sync.WaitGroup{}

	for b.Loop() {
		b.StopTimer()
		shardName := consts.DefaultQueueShardName
		r := miniredis.RunT(b)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		rc = redis_telemetry.InstrumentRedisClient(context.Background(), rc, redis_telemetry.InstrumentedClientOpts{
			PkgName: "benchmark",
			Cluster: shardName,
		})
		require.NoError(b, err)
		defer rc.Close()

		clock := clockwork.NewRealClock()

		defaultShard := redis_state.QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey), Name: shardName}

		var q osqueue.Queue = redis_state.NewQueue(
			defaultShard,
			redis_state.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			redis_state.WithEnqueueSystemPartitionsToBacklog(false),
			redis_state.WithDisableLeaseChecksForSystemQueues(false),
			redis_state.WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return false
			}),
			// system queue items
			redis_state.WithKindToQueueMapping(map[string]string{
				osqueue.KindScheduleBatch:   osqueue.KindScheduleBatch,
				osqueue.KindDebounce:        osqueue.KindDebounce,
				osqueue.KindQueueMigrate:    osqueue.KindQueueMigrate,
				osqueue.KindPauseBlockFlush: osqueue.KindPauseBlockFlush,
				osqueue.KindJobPromote:      osqueue.KindJobPromote,
			}),
			redis_state.WithBacklogRefillLimit(10),
			redis_state.WithClock(clock),
			redis_state.WithRunMode(redis_state.QueueRunMode{
				Sequential:                        true,
				Scavenger:                         true,
				Partition:                         true,
				Account:                           true,
				AccountWeight:                     80,
				Continuations:                     true,
				ShadowPartition:                   true,
				AccountShadowPartition:            true,
				AccountShadowPartitionWeight:      80,
				ShadowContinuations:               true,
				ShadowContinuationSkipProbability: consts.QueueContinuationSkipProbability,
				NormalizePartition:                true,
				ActiveChecker:                     true,
			}),
			redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.QueueShard, error) {
				return defaultShard, nil
			}),
		)

		accountID, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		var counter int64

		withTimeout, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()

		eg := errgroup.Group{}
		eg.Go(func() error {
			return q.Run(withTimeout, func(ctx context.Context, info osqueue.RunInfo, item osqueue.Item) (osqueue.RunResult, error) {
				defer wg.Done()

				if progress := int(atomic.LoadInt64(&counter)); progress%(processItems/10) == 0 {
					fmt.Println("progress: ", progress)
				}

				// TODO instrument these
				// info.Latency
				// info.ContinueCount
				// info.SojournDelay

				atomic.AddInt64(&counter, 1)

				return osqueue.RunResult{
					ScheduledImmediateJob: false,
				}, nil
			})
		})
		b.StartTimer()

		for range processItems {
			wg.Add(1)
			err = q.Enqueue(ctx, osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:      fnID,
					WorkflowVersion: 1,
					AccountID:       accountID,
					WorkspaceID:     wsID,
				},
			}, clock.Now(), osqueue.EnqueueOpts{})
			require.NoError(b, err)
		}

		wg.Wait()
		fmt.Println("all processed")

		cancel()
		err = eg.Wait()
		if err != nil {
			require.ErrorIs(b, err, context.Canceled)
		}
	}

}
