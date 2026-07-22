package metrics

import "context"

// IncrEventSessionsResolvedCounter records one session resolution (one call to
// event.EventMeta.ResolveSessions), tagged by source and by whether each layer
// was present. It is emitted on every resolve — including the neither-present
// case — so the {manual=false,propagated=false} series is a self-contained
// denominator: manual-set rate = sum(manual=true)/sum(all), propagated rate =
// sum(propagated=true)/sum(all).
//
// The server merge is the single vantage point that sees both layers for every
// spawn primitive and SDK version. Tags are low-cardinality only (source, two
// bools); per-account/per-session detail belongs in the analytics plane
// (Insights runs.sessions), never here.
func IncrEventSessionsResolvedCounter(ctx context.Context, source string, manual, propagated bool, opts CounterOpt) {
	if opts.Tags == nil {
		opts.Tags = map[string]any{}
	}
	opts.Tags["source"] = source
	opts.Tags["manual"] = manual
	opts.Tags["propagated"] = propagated

	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "event_sessions_resolved_total",
		Description: "Total event session resolutions, tagged by source and which layers were present",
		Tags:        opts.Tags,
	})
}
