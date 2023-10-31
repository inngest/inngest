package store

import (
	"time"
)

// Option represents a store option function.
type Option func(o *Options)

type Options struct {
	Cost                      int64
	Expiration                time.Duration
	Tags                      []string
	ClientSideCacheExpiration time.Duration
}

func (o *Options) IsEmpty() bool {
	return o.Cost == 0 && o.Expiration == 0 && len(o.Tags) == 0
}

func ApplyOptionsWithDefault(defaultOptions *Options, opts ...Option) *Options {
	returnedOptions := &Options{}
	*returnedOptions = *defaultOptions

	for _, opt := range opts {
		opt(returnedOptions)
	}

	return returnedOptions
}

func ApplyOptions(opts ...Option) *Options {
	o := &Options{}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

// WithCost allows setting the memory capacity used by the item when setting a value.
// Actually it seems to be used by Ristretto library only.
func WithCost(cost int64) Option {
	return func(o *Options) {
		o.Cost = cost
	}
}

// WithExpiration allows to specify an expiration time when setting a value.
func WithExpiration(expiration time.Duration) Option {
	return func(o *Options) {
		o.Expiration = expiration
	}
}

// WithTags allows to specify associated tags to the current value.
func WithTags(tags []string) Option {
	return func(o *Options) {
		o.Tags = tags
	}
}

// WithClientSideCaching allows setting the client side caching, enabled by default
// Currently to be used by Rueidis(redis) library only.
func WithClientSideCaching(clientSideCacheExpiration time.Duration) Option {
	return func(o *Options) {
		o.ClientSideCacheExpiration = clientSideCacheExpiration
	}
}
