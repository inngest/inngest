package metrics

import "context"

// IncrEventSessionsResolvedCounter records one session resolution (one call to
// event.EventMeta.ResolveSessions), tagged by source and by the pre-merge state
// of the two session layers. It is emitted on every resolve — including the
// nothing-present case — so the {manual=false,propagated=false,nulling=false}
// series is a self-contained denominator: manual-set rate =
// sum(manual=true)/sum(all), propagated rate = sum(propagated=true)/sum(all).
//
// manual and nulling are independent: manual covers concrete keys the user set,
// nulling covers JSON nulls used to suppress inherited sessions (whole-field or
// per-key). Any manual interaction at all is manual=true OR nulling=true;
// clear-and-set within one event — the signal for whether a combined operation
// is worth having — is manual=true AND nulling=true.
//
// Callers pass the fields of an event.SessionsMetrics. The struct itself is
// not taken directly because pkg/event depends on this package, so the import
// cannot run the other way.
//
// The server merge is the single vantage point that sees both layers for every
// spawn primitive and SDK version. Tags are low-cardinality only (source, three
// bools); per-account/per-session detail belongs in the analytics plane
// (Insights runs.sessions), never here.
func IncrEventSessionsResolvedCounter(ctx context.Context, source string, manual, propagated, nulling bool, opts CounterOpt) {
	if opts.Tags == nil {
		opts.Tags = map[string]any{}
	}
	opts.Tags["source"] = source
	opts.Tags["manual"] = manual
	opts.Tags["propagated"] = propagated
	opts.Tags["nulling"] = nulling

	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "event_sessions_resolved_total",
		Description: "Total event session resolutions, tagged by source and the pre-merge state of each session layer",
		Tags:        opts.Tags,
	})
}
