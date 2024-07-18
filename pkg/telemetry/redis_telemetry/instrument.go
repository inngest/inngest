package redis_telemetry

import (
	"context"
	"github.com/inngest/inngest/pkg/telemetry"
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
	pkgName string
	cluster string
	rueidis.Client
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

	telemetry.HistogramRedisCommandDuration(ctx, dur.Milliseconds(), telemetry.HistogramOpt{
		PkgName: i.pkgName,
		Tags:    tags,
	})
}

func (i instrumentedClient) asyncReport(ctx context.Context, start time.Time, command string) {
	end := time.Now()

	go i.report(ctx, start, end, command)
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

type InstrumentedClientOpts struct {
	PkgName string
	Cluster string
}

func wrapWithObservability(c rueidis.Client, opts InstrumentedClientOpts) rueidis.Client {
	return &instrumentedClient{opts.PkgName, opts.Cluster, c}
}
