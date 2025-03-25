package redis_telemetry

import (
	"context"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/redis/rueidis"
	"time"
)

type scopeValType struct{}
type scriptNameValType struct{}
type opNameCtxValType struct{}

var (
	scopeCtxVal      = scopeValType{}
	scriptNameCtxVal = scriptNameValType{}
	opNameCtxVal     = opNameCtxValType{}
)

type Scope string

const (
	ScopeQueue      Scope = "queue"
	ScopePauses     Scope = "pauses"
	ScopeFnRunState Scope = "state"
)

// WithOpName returns a context that stores the given opName inside.
func WithOpName(ctx context.Context, opName string) context.Context {
	return context.WithValue(ctx, opNameCtxVal, opName)
}

// WithScope returns a context that stores the given scope inside.
func WithScope(ctx context.Context, scope Scope) context.Context {
	return context.WithValue(ctx, scopeCtxVal, scope)
}

// WithScope returns a context that stores the given scope inside.
func WithScriptName(ctx context.Context, scriptName string) context.Context {
	return context.WithValue(ctx, scriptNameCtxVal, scriptName)
}

// scopeFromContext returns the scope given the current context, or an
// empty string if there's no scope.
func scopeFromContext(ctx context.Context) Scope {
	str, _ := ctx.Value(scopeCtxVal).(Scope)
	return str
}

// scriptNameFromContext returns the scope given the current context, or an
// empty string if there's no scope.
func scriptNameFromContext(ctx context.Context) string {
	str, _ := ctx.Value(scriptNameCtxVal).(string)
	return str
}

// opNameFromContext returns the scope given the current context, or an
// empty string if there's no scope.
func opNameFromContext(ctx context.Context) string {
	str, _ := ctx.Value(opNameCtxVal).(string)
	return str
}

type instrumentedClient struct {
	reports chan *reportItem

	pkgName string
	cluster string
	rueidis.Client
}

func (i instrumentedClient) Do(ctx context.Context, cmd rueidis.Completed) (resp rueidis.RedisResult) {
	start := time.Now()

	command := ""
	if len(cmd.Commands()) > 0 {
		command = cmd.Commands()[0]
	}

	defer i.asyncReport(ctx, start, command)

	return i.Client.Do(ctx, cmd)
}

type reportItem struct {
	ctx     context.Context
	start   time.Time
	end     time.Time
	command string
}

const defaultBufferSize int = 100_000
const defaultNumWorkers int = 10_000

type InstrumentedClientOpts struct {
	PkgName string
	Cluster string

	BufferSize int
	NumWorkers int
}

func InstrumentRedisClient(ctx context.Context, c rueidis.Client, opts InstrumentedClientOpts) rueidis.Client {
	numWorkers := opts.NumWorkers
	if numWorkers == 0 {
		numWorkers = defaultNumWorkers
	}

	bufferSize := opts.BufferSize
	if bufferSize == 0 {
		bufferSize = defaultBufferSize
	}

	reports := make(chan *reportItem, bufferSize)

	instrumented := &instrumentedClient{reports, opts.PkgName, opts.Cluster, c}

	for i := 0; i < numWorkers; i++ {
		go instrumented.worker(ctx)
	}

	return instrumented
}

func (i instrumentedClient) report(ctx context.Context, start, end time.Time, command string) {
	dur := end.Sub(start)

	tags := map[string]any{
		"cluster": i.cluster,
	}
	if command != "" {
		tags["command"] = command
	}

	scope := scopeFromContext(ctx)
	if scope != "" {
		tags["scope"] = string(scope)
	}

	scriptName := scriptNameFromContext(ctx)
	if scriptName != "" {
		tags["script_name"] = scriptName
	}

	opName := opNameFromContext(ctx)
	if opName != "" {
		tags["op"] = opName
	}

	metrics.HistogramRedisCommandDuration(ctx, dur.Milliseconds(), metrics.HistogramOpt{
		PkgName: i.pkgName,
		Tags:    tags,
	})
}

func (i instrumentedClient) asyncReport(ctx context.Context, start time.Time, command string) {
	end := time.Now()

	i.reports <- &reportItem{ctx, start, end, command}
}

func (i instrumentedClient) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-i.reports:
			i.report(item.ctx, item.start, item.end, item.command)
		}
	}
}
