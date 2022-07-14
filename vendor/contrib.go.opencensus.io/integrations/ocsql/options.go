package ocsql

import (
	"go.opencensus.io/trace"
)

// TraceOption allows for managing ocsql configuration using functional options.
type TraceOption func(o *TraceOptions)

const defaultInstanceName = "default"

// TraceOptions holds configuration of our ocsql tracing middleware.
// By default all options are set to false intentionally when creating a wrapped
// driver and provide the most sensible default with both performance and
// security in mind.
type TraceOptions struct {
	// AllowRoot, if set to true, will allow ocsql to create root spans in
	// absence of existing spans or even context.
	// Default is to not trace ocsql calls if no existing parent span is found
	// in context or when using methods not taking context.
	AllowRoot bool

	// Ping, if set to true, will enable the creation of spans on Ping requests.
	Ping bool

	// RowsNext, if set to true, will enable the creation of spans on RowsNext
	// calls. This can result in many spans.
	RowsNext bool

	// RowsClose, if set to true, will enable the creation of spans on RowsClose
	// calls.
	RowsClose bool

	// RowsAffected, if set to true, will enable the creation of spans on
	// RowsAffected calls.
	RowsAffected bool

	// LastInsertID, if set to true, will enable the creation of spans on
	// LastInsertId calls.
	LastInsertID bool

	// Query, if set to true, will enable recording of sql queries in spans.
	// Only allow this if it is safe to have queries recorded with respect to
	// security.
	Query bool

	// QueryParams, if set to true, will enable recording of parameters used
	// with parametrized queries. Only allow this if it is safe to have
	// parameters recorded with respect to security.
	// This setting is a noop if the Query option is set to false.
	QueryParams bool

	// DefaultAttributes will be set to each span as default.
	DefaultAttributes []trace.Attribute

	// InstanceName identifies database.
	InstanceName string

	// DisableErrSkip, if set to true, will suppress driver.ErrSkip errors in spans.
	DisableErrSkip bool

	// Sampler to use when creating spans.
	Sampler trace.Sampler
}

// WithAllTraceOptions enables all available trace options.
func WithAllTraceOptions() TraceOption {
	return func(o *TraceOptions) {
		*o = AllTraceOptions
	}
}

// AllTraceOptions has all tracing options enabled.
var AllTraceOptions = TraceOptions{
	AllowRoot:    true,
	Ping:         true,
	RowsNext:     true,
	RowsClose:    true,
	RowsAffected: true,
	LastInsertID: true,
	Query:        true,
	QueryParams:  true,
}

// WithOptions sets our ocsql tracing middleware options through a single
// TraceOptions object.
func WithOptions(options TraceOptions) TraceOption {
	return func(o *TraceOptions) {
		*o = options
		o.DefaultAttributes = append(
			[]trace.Attribute(nil), options.DefaultAttributes...,
		)
	}
}

// WithAllowRoot if set to true, will allow ocsql to create root spans in
// absence of exisiting spans or even context.
// Default is to not trace ocsql calls if no existing parent span is found
// in context or when using methods not taking context.
func WithAllowRoot(b bool) TraceOption {
	return func(o *TraceOptions) {
		o.AllowRoot = b
	}
}

// WithPing if set to true, will enable the creation of spans on Ping requests.
func WithPing(b bool) TraceOption {
	return func(o *TraceOptions) {
		o.Ping = b
	}
}

// WithRowsNext if set to true, will enable the creation of spans on RowsNext
// calls. This can result in many spans.
func WithRowsNext(b bool) TraceOption {
	return func(o *TraceOptions) {
		o.RowsNext = b
	}
}

// WithRowsClose if set to true, will enable the creation of spans on RowsClose
// calls.
func WithRowsClose(b bool) TraceOption {
	return func(o *TraceOptions) {
		o.RowsClose = b
	}
}

// WithRowsAffected if set to true, will enable the creation of spans on
// RowsAffected calls.
func WithRowsAffected(b bool) TraceOption {
	return func(o *TraceOptions) {
		o.RowsAffected = b
	}
}

// WithLastInsertID if set to true, will enable the creation of spans on
// LastInsertId calls.
func WithLastInsertID(b bool) TraceOption {
	return func(o *TraceOptions) {
		o.LastInsertID = b
	}
}

// WithQuery if set to true, will enable recording of sql queries in spans.
// Only allow this if it is safe to have queries recorded with respect to
// security.
func WithQuery(b bool) TraceOption {
	return func(o *TraceOptions) {
		o.Query = b
	}
}

// WithQueryParams if set to true, will enable recording of parameters used
// with parametrized queries. Only allow this if it is safe to have
// parameters recorded with respect to security.
// This setting is a noop if the Query option is set to false.
func WithQueryParams(b bool) TraceOption {
	return func(o *TraceOptions) {
		o.QueryParams = b
	}
}

// WithDefaultAttributes will be set to each span as default.
func WithDefaultAttributes(attrs ...trace.Attribute) TraceOption {
	return func(o *TraceOptions) {
		o.DefaultAttributes = attrs
	}
}

// WithDisableErrSkip, if set to true, will suppress driver.ErrSkip errors in spans.
func WithDisableErrSkip(b bool) TraceOption {
	return func(o *TraceOptions) {
		o.DisableErrSkip = b
	}
}

// WithSampler will be used on span creation.
func WithSampler(sampler trace.Sampler) TraceOption {
	return func(o *TraceOptions) {
		o.Sampler = sampler
	}
}

// WithInstanceName sets database instance name.
func WithInstanceName(instanceName string) TraceOption {
	return func(o *TraceOptions) {
		o.InstanceName = instanceName
	}
}
